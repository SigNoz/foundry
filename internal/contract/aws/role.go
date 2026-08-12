package aws

// Role is what a security group or an IAM role is attached to. A security group
// uses node and task; the rest are IAM only, and cluster is the one a managed
// control plane assumes rather than a workload.
type Role struct {
	s string
}

var (
	RoleNode    = Role{s: "node"}
	RoleTask    = Role{s: "task"}
	RoleExec    = Role{s: "exec"}
	RoleCluster = Role{s: "cluster"}
	RoleEBSCSI  = Role{s: "ebs-csi"}
)

func (role Role) String() string {
	return role.s
}
