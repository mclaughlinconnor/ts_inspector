package parser

import (
	"sync"
	"ts_inspector/utils"
)

type State struct {
	sync.RWMutex
	classes map[string]*Class
	files   map[string]*File
	rootURI string
}

func CreateState(projectRoot string) State {
	return State{classes: map[string]*Class{}, files: map[string]*File{}, rootURI: projectRoot}
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

func (s *State) GetRootUri() *string {
	s.RLock()
	rootUri := &s.rootURI
	s.RUnlock()

	return rootUri
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
