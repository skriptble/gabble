package bosh

type Version struct {
	Major, Minor int
}

// Compare takes a version and returns the version with the lower version
// number.
func (v Version) Compare(o Version) Version {
	if v.Major < o.Major {
		return v
	}
	if v.Major > o.Major {
		return o
	}

	if v.Minor < o.Minor {
		return v
	}

	return o

}
