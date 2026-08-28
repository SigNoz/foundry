package mechanic

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strings"

	"github.com/signoz/foundry/api/v1alpha1"
	"github.com/signoz/foundry/api/v1alpha1/installation"
	"github.com/signoz/foundry/internal/domain"
	"github.com/signoz/foundry/internal/errors"
)

// ruleQuery looks up a single alert rule by id. id is interpolated as a literal
// because the store CLIs (psql, sqlite3) are driven as text with no driver to
// parameterize; safeID guards the value before it reaches here.
const ruleQuery = "SELECT id, data FROM rule WHERE id = '%s'"

// fieldSep is an ASCII unit separator used to split columns in store CLI
// output. Unlike a comma or pipe it will not collide with the JSON payload in
// the data column.
const fieldSep = "\x1f"

// sqliteDefaultPath is where SigNoz keeps its embedded sqlite database inside
// the signoz container when SIGNOZ_SQLSTORE_SQLITE_PATH is absent from the lock.
const sqliteDefaultPath = "/var/lib/signoz/signoz.db"

// metastorePostgresCredential is the value the metastore molding provisions for
// the postgres user, database, and password alike (all three are "signoz"). See
// internal/molding/metastoremolding.
const metastorePostgresCredential = "signoz"

// safeID restricts ids interpolated into SQL to the UUID/integer alphabet
// SigNoz uses for rule ids, closing off injection via the text-driven CLIs.
var safeID = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

// Executor runs an external command and returns its stdout. Abstracted so the
// store reachers can be exercised without a live deployment.
type Executor interface {
	Output(ctx context.Context, name string, args ...string) ([]byte, error)
}

// MetaStore reads SigNoz state from a deployment's metadata store.
type MetaStore interface {
	// Rule returns the alert rule stored under id.
	Rule(ctx context.Context, id string) (Alert, error)
}

// NewMetaStore selects the reach strategy for the deployment's metastore. Phase
// 1 supports docker/compose only by executing into the running store container;
// other targets return TypeUnsupported.
func NewMetaStore(executor Executor, machinery v1alpha1.Machinery) (MetaStore, error) {
	c, err := dockerComposeCasting(machinery)
	if err != nil {
		return nil, err
	}

	switch c.Spec.MetaStore.Kind {
	case installation.MetaStoreKindPostgres:
		container, err := firstHost(c.Spec.MetaStore.Status.Addresses.DSN)
		if err != nil {
			return nil, errors.Wrapf(err, errors.TypeNotFound, "metastore address missing from lock, run forge first")
		}
		return &dockerPostgresMetaStore{executor: executor, container: container}, nil

	case installation.MetaStoreKindSQLite:
		container, err := firstHost(c.Spec.Signoz.Status.Addresses.APIServer)
		if err != nil {
			return nil, errors.Wrapf(err, errors.TypeNotFound, "signoz address missing from lock, run forge first")
		}
		path := c.Spec.Signoz.Spec.Env["SIGNOZ_SQLSTORE_SQLITE_PATH"]
		if path == "" {
			path = sqliteDefaultPath
		}
		return &dockerSQLiteMetaStore{executor: executor, container: container, path: path}, nil

	default:
		return nil, errors.Newf(errors.TypeUnsupported, "unsupported metastore kind %q", c.Spec.MetaStore.Kind)
	}
}

// dockerPostgresMetaStore reaches a postgres metastore by exec-ing psql inside
// the running postgres container. Credentials are the fixed signoz/signoz/signoz
// trio the metastore molding provisions.
type dockerPostgresMetaStore struct {
	executor  Executor
	container string
}

func (m *dockerPostgresMetaStore) Rule(ctx context.Context, id string) (Alert, error) {
	if !safeID.MatchString(id) {
		return Alert{}, errors.Newf(errors.TypeInvalidInput, "invalid alert id %q", id)
	}

	out, err := m.executor.Output(ctx, "docker", "exec",
		"-e", "PGPASSWORD="+metastorePostgresCredential, m.container,
		"psql", "-U", metastorePostgresCredential, "-d", metastorePostgresCredential,
		"-tA", "-F", fieldSep,
		"-c", fmt.Sprintf(ruleQuery, id),
	)
	if err != nil {
		return Alert{}, errors.Wrapf(err, errors.TypeInternal, "failed to query metastore via container %q", m.container)
	}

	return parseRuleRow(out, id)
}

// dockerSQLiteMetaStore reaches an embedded sqlite metastore by exec-ing sqlite3
// against the database file inside the running signoz container.
type dockerSQLiteMetaStore struct {
	executor  Executor
	container string
	path      string
}

func (m *dockerSQLiteMetaStore) Rule(ctx context.Context, id string) (Alert, error) {
	if !safeID.MatchString(id) {
		return Alert{}, errors.Newf(errors.TypeInvalidInput, "invalid alert id %q", id)
	}

	// The signoz image ships without a sqlite3 client, so install it (as root,
	// since signoz may run unprivileged) before querying. apk add is idempotent,
	// and running it as its own exec keeps its output out of the query parse.
	if _, err := m.executor.Output(ctx, "docker", "exec", "-u", "root", m.container,
		"apk", "add", "--no-cache", "sqlite",
	); err != nil {
		return Alert{}, errors.Wrapf(err, errors.TypeInternal, "failed to install sqlite3 in container %q", m.container)
	}

	out, err := m.executor.Output(ctx, "docker", "exec", m.container,
		"sqlite3", "-separator", fieldSep, m.path,
		fmt.Sprintf(ruleQuery, id),
	)
	if err != nil {
		return Alert{}, errors.Wrapf(err, errors.TypeInternal, "failed to query sqlite metastore via container %q", m.container)
	}

	return parseRuleRow(out, id)
}

// parseRuleRow decodes the first <id><sep><data> row of a store CLI's output.
// An empty result means no rule matched the id.
func parseRuleRow(out []byte, id string) (Alert, error) {
	row := strings.TrimSpace(string(out))
	if row == "" {
		return Alert{}, errors.Newf(errors.TypeNotFound, "no alert found with id %q", id)
	}

	if idx := strings.IndexByte(row, '\n'); idx >= 0 {
		row = row[:idx]
	}

	col, data, ok := strings.Cut(row, fieldSep)
	if !ok {
		return Alert{}, errors.Newf(errors.TypeInternal, "unexpected metastore row format for id %q", id)
	}

	return decodeAlert(strings.TrimSpace(col), []byte(strings.TrimSpace(data))), nil
}

// firstHost returns the host of the first address, the container name to exec
// into for docker/compose deployments.
func firstHost(addresses []string) (string, error) {
	if len(addresses) == 0 {
		return "", errors.Newf(errors.TypeNotFound, "no address available")
	}

	addr, err := domain.ParseAddress(addresses[0])
	if err != nil {
		return "", err
	}

	return addr.Host(), nil
}

// execExecutor is the production Executor backed by os/exec.
type execExecutor struct{}

// NewExecExecutor returns an Executor that shells out via os/exec.
func NewExecExecutor() Executor {
	return execExecutor{}
}

func (execExecutor) Output(ctx context.Context, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if msg := strings.TrimSpace(stderr.String()); msg != "" {
			return nil, errors.Wrapf(err, errors.TypeInternal, "%s", msg)
		}
		return nil, err
	}

	return stdout.Bytes(), nil
}
