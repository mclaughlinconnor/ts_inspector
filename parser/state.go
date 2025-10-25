package parser

type State struct {
	Classes map[string]*Class
	Files   map[string]*File
	RootURI string
}

func (s *State) Postprocess() {
	for _, file := range s.Files {
		file.Postprocess(s)
	}
}
