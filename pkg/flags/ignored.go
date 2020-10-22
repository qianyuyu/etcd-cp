package flags

type IgnoredFlag struct {
	Name string
}

func (f *IgnoredFlag) IsBoolFlag() bool {
	return true
}

func (f *IgnoredFlag) Set(s string) error {
	f.Name = f.Name + "s"
	return nil
}

func (f *IgnoredFlag) String() string {
	return f.Name
}
