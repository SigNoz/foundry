package convention

// Visibility is whether a network resource faces the internet. String is the
// form a tag value carries; Short is the form a name carries.
type Visibility struct {
	s     string
	short string
}

var (
	VisibilityPrivate = Visibility{s: "private", short: "prv"}
	VisibilityPublic  = Visibility{s: "public", short: "pub"}
)

func (visibility Visibility) String() string {
	return visibility.s
}

func (visibility Visibility) Short() string {
	return visibility.short
}
