package convention

// Role is what a security group or an IAM role is attached to. IAM roles use all
// three; a security group uses node and task.
type Role struct {
	s string
}

var (
	RoleNode = Role{s: "node"}
	RoleTask = Role{s: "task"}
	RoleExec = Role{s: "exec"}
)

func (role Role) String() string {
	return role.s
}
