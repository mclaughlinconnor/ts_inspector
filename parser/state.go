package parser

import (
	"log"
	"sync"
	"ts_inspector/utils"

	sitter "github.com/smacker/go-tree-sitter"
)

// Hacky "fix" because of my badly structured packages
type TcbGeneratorFunc func(state *State, class *Class, root *sitter.Node, content []byte) string

type State struct {
	sync.RWMutex

	Logger *log.Logger

	classes       map[string]*Class
	files         map[string]*File
	rootURI       string
	tcbGenerator  TcbGeneratorFunc
	tsConfigFiles []string
	tsgo          *TsGo
}

func CreateState() State {
	return State{
		Logger:  utils.GetLogger("ts_inspector"),
		classes: map[string]*Class{},
		files:   map[string]*File{},
	}
}

func (s *State) GetFile(filename string) (*File, bool) {
	s.RLock()
	file, found := s.files[filename]
	s.RUnlock()

	return file, found
}

func (s *State) GetFiles() *map[string]*File {
	s.RLock()
	files := &s.files
	s.RUnlock()

	return files
}

func (s *State) GetClass(id string) (*Class, bool) {
	s.RLock()
	class, found := s.classes[id]
	s.RUnlock()

	return class, found
}

func (s *State) GetClasses() *map[string]*Class {
	s.RLock()
	classes := &s.classes
	s.RUnlock()

	return classes
}

func (s *State) GetClassesBySelectorUsage(selectors []string) []*Class {
	classes := []*Class{}

	s.RLock()

	for _, selector := range selectors {
		for _, class := range *s.GetClasses() {
			if !class.HasComponent() || class.Snapshot().Angular.Component.Template == nil {
				continue
			}

			tagUsages := class.Snapshot().Angular.Component.Template.TagUsages
			_, found := tagUsages[selector]
			if !found {
				continue
			}

			classes = append(classes, class)
		}
	}

	s.RUnlock()

	return classes
}

func (s *State) GetInterestingPoints() []InterestingPoint {
	interestingPoints := make([]InterestingPoint, 0)
	for _, class := range *s.GetClasses() {
		interestingPoints = append(interestingPoints, class.GetInterestingPoints()...)
	}

	for _, file := range *s.GetFiles() {
		interestingPoints = append(interestingPoints, file.GetInterestingPoints()...)
	}

	return interestingPoints
}

func (s *State) GetRootPath() string {
	s.RLock()
	rootPath := FilenameFromUri(s.rootURI)
	s.RUnlock()

	return rootPath
}

func (s *State) GetTcbGenerator() TcbGeneratorFunc {
	s.RLock()
	gen := s.tcbGenerator
	s.RUnlock()

	return gen
}

func (s *State) GetTsConfigFiles() []string {
	s.RLock()
	files := s.tsConfigFiles
	s.RUnlock()

	return files
}

func (s *State) GetTsGo() *TsGo {
	if !utils.TsGo {
		s.Logger.Fatalln("You may not call \"GetTsGo\" when the TsGo integration is disabled")
	}

	s.RLock()
	tsgo := s.tsgo
	s.RUnlock()

	return tsgo
}

func (s *State) Postprocess() {
	wg := sync.WaitGroup{}

	for _, file := range *s.GetFiles() {
		wg.Go(func() { file.Postprocess(s) })

		if !utils.Concurrency {
			wg.Wait()
		}
	}

	wg.Wait()
}

func (s *State) SetClass(id string, class *Class) {
	s.Lock()
	s.classes[id] = class
	s.Unlock()
}

func (s *State) SetFile(filename string, file *File) {
	s.Lock()
	s.files[filename] = file
	s.Unlock()
}

func (s *State) SetRootUri(rootUri string) {
	s.Lock()
	s.rootURI = rootUri
	s.Unlock()
}

func (s *State) SetTcbGenerator(gen TcbGeneratorFunc) {
	s.Lock()
	s.tcbGenerator = gen
	s.Unlock()
}

func (s *State) SetTsConfigFiles(tsconfigFiles []string) {
	s.Lock()
	s.tsConfigFiles = tsconfigFiles
	s.Unlock()
}

func (s *State) SetTsGo(tsgo *TsGo) {
	s.Lock()
	s.tsgo = tsgo
	s.Unlock()
}
