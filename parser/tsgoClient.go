package parser

import (
	"bufio"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log"
	"os/exec"
	"sync"
	"ts_inspector/config"
	"ts_inspector/rpc"
	"ts_inspector/utils"
)

type Responses struct {
	sync.RWMutex

	response map[string]chan []byte
}

type TsGo struct {
	logger         *log.Logger
	opLock         sync.Mutex
	projectHandle  ProjectID
	responses      *Responses
	rootFiles      map[string]bool
	snapshotHandle SnapshotID
	state          *State

	stderr *io.ReadCloser
	stdin  *utils.Writer
	stdout *io.ReadCloser

	ctx    context.Context
	cancel context.CancelFunc
}

func StartTsGo(state *State) (*TsGo, error) {
	args := []string{
		config.GetConfig().TsGo.BinaryPath,
		"--api",
		"--async",
		"--callbacks",
		"readFile,fileExists,getAccessibleEntries",
		"--cwd",
		state.GetRootPath(),
	}

	cmd := exec.Command("node", args...)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, err
	}

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	stdinWriter := utils.NewWriter(stdin)

	ctx, cancel := context.WithCancel(context.Background())

	tsgo := &TsGo{
		logger:    state.Logger,
		responses: &Responses{response: make(map[string]chan []byte)},
		rootFiles: make(map[string]bool),
		state:     state,

		stderr: &stderr,
		stdin:  stdinWriter,
		stdout: &stdout,

		cancel: cancel,
		ctx:    ctx,
	}

	go run(tsgo)
	go stderrLogger(tsgo)

	return tsgo, nil
}

func run(tsgo *TsGo) {
	defer utils.PanicLogger(tsgo.logger)

	scanner := bufio.NewScanner(*tsgo.stdout)

	scanner.Split(rpc.Split)
	big := 1024 * 1024 // 1 mb
	buf := make([]byte, big)
	scanner.Buffer(buf, big)

	for scanner.Scan() {
		msg := scanner.Bytes()
		_, contents, err := rpc.DecodeMessage(msg)
		if err != nil {
			continue
		}

		r := utils.TryParseRequest[TsGoRequest](tsgo.logger, contents)
		if r.Method == "" {
			tsgo.responses.GetHandler(r.ID) <- contents
		} else {
			tsgo.handleRequest(r.Method, contents)
		}
	}
}

func stderrLogger(tsgo *TsGo) {
	scanner := bufio.NewScanner(*tsgo.stderr)

	big := 1024 * 1024 // 1 mb
	buf := make([]byte, big)
	scanner.Buffer(buf, big)

	for scanner.Scan() {
		msg := scanner.Bytes()
		tsgo.logger.Println(string(msg))
	}

	err := scanner.Err()
	if err != nil {
		tsgo.logger.Println(err)
	}
}

func (r *Responses) AddHandler(id string, channel chan []byte) {
	r.Lock()
	r.response[id] = channel
	r.Unlock()
}

func (r *Responses) GetHandler(id string) chan []byte {
	r.RLock()
	c := r.response[id]
	r.RUnlock()

	return c
}

func (t *TsGo) GetSemanticDiagnostics(uri string) *DiagnosticResponse {
	t.opLock.Lock()
	defer t.opLock.Unlock()

	t.updateSnapshotLocked(nil, []DocumentIdentifier{{URI: uri}})

	id := utils.GetNextStringId()
	request := GetDiagnosticsRequest{
		TsGoRequest: TsGoRequest{RPC: "2.0", ID: id, Method: "getSemanticDiagnostics"},
		Params: GetDiagnosticsParams{
			Snapshot: t.snapshotHandle,
			Project:  t.projectHandle,
			File:     &DocumentIdentifier{URI: uri},     // old
			Files:    &[]DocumentIdentifier{{URI: uri}}, // new
		},
	}

	utils.WriteResponse(t.stdin, request)

	c := make(chan []byte, 1)
	t.responses.AddHandler(id, c)

	select {
	case <-t.ctx.Done():
		return nil
	case result := <-c:
		r := utils.TryParseRequest[DiagnosticResponse](t.logger, result)
		return &r
	}
}

func (t *TsGo) GetSymbolAtPosition(uri string, offset int) *Symbol {
	t.opLock.Lock()
	defer t.opLock.Unlock()

	t.updateSnapshotLocked(nil, []DocumentIdentifier{{URI: uri}})

	id := utils.GetNextStringId()
	request := GetSymbolAtPositionRequest{
		TsGoRequest: TsGoRequest{RPC: "2.0", ID: id, Method: "getSymbolAtPosition"},
		Params: GetSymbolAtPositionParams{
			File:     DocumentIdentifier{URI: uri},
			Position: uint32(offset),
			Project:  t.projectHandle,
			Snapshot: t.snapshotHandle,
		},
	}

	utils.WriteResponse(t.stdin, request)

	c := make(chan []byte, 1)
	t.responses.AddHandler(id, c)

	select {
	case <-t.ctx.Done():
		return nil
	case result := <-c:
		r := utils.TryParseRequest[SymbolResponse](t.logger, result)
		return &r.Result
	}
}

func (t *TsGo) GetTypeOfSymbol(symbol SymbolID) *TypeResponse {
	t.opLock.Lock()
	defer t.opLock.Unlock()
	id := utils.GetNextStringId()
	request := GetTypeOfSymbolRequest{
		TsGoRequest: TsGoRequest{RPC: "2.0", ID: id, Method: "getTypeOfSymbol"},
		Params: GetTypeOfSymbolParams{
			Project:  t.projectHandle,
			Snapshot: t.snapshotHandle,
			Symbol:   symbol,
		},
	}

	utils.WriteResponse(t.stdin, request)

	c := make(chan []byte, 1)
	t.responses.AddHandler(id, c)

	select {
	case <-t.ctx.Done():
		return nil
	case result := <-c:
		r := utils.TryParseRequest[TypeResponse](t.logger, result)
		return &r
	}
}

func (t *TsGo) GetTypeAtPosition(uri string, offset int) *TypeResponse {
	documentIdentifier := DocumentIdentifier{URI: uri}
	t.UpdateSnapshot("", []DocumentIdentifier{documentIdentifier})

	id := utils.GetNextStringId()
	request := GetTypeAtPositionParamsRequest{
		TsGoRequest: TsGoRequest{RPC: "2.0", ID: id, Method: "getTypeAtPosition"},
		Params: GetTypeAtPositionParams{
			Project:  t.projectHandle,
			Snapshot: t.snapshotHandle,
			File:     documentIdentifier,
			Position: uint32(offset),
		},
	}

	utils.WriteResponse(t.stdin, request)

	c := make(chan []byte, 1)
	t.responses.AddHandler(id, c)

	select {
	case <-t.ctx.Done():
		return nil
	case result := <-c:
		r := utils.TryParseRequest[TypeResponse](t.logger, result)
		return &r
	}
}

func (t *TsGo) Initialize() *InitializeResponse {
	t.opLock.Lock()
	defer t.opLock.Unlock()

	id := utils.GetNextStringId()
	request := TsGoRequest{RPC: "2.0", ID: id, Method: "initialize"}

	utils.WriteResponse(t.stdin, request)

	c := make(chan []byte, 1)
	t.responses.AddHandler(id, c)

	select {
	case <-t.ctx.Done():
		return nil
	case result := <-c:
		r := utils.TryParseRequest[InitializeResponse](t.logger, result)
		return &r
	}
}

func (t *TsGo) TypeToString(ttype TypeID) *TypeToStringResponse {
	t.opLock.Lock()
	defer t.opLock.Unlock()
	id := utils.GetNextStringId()
	request := TypeToTypeNodeRequest{
		TsGoRequest: TsGoRequest{RPC: "2.0", ID: id, Method: "typeToString"},
		Params: TypeToTypeNodeParams{
			Flags:    TypeFormatFlagsUseAliasDefinedOutsideCurrentScope | TypeFormatFlagsUseInstantiationExpressions,
			Project:  t.projectHandle,
			Snapshot: t.snapshotHandle,
			Type:     ttype,
		},
	}

	utils.WriteResponse(t.stdin, request)

	c := make(chan []byte, 1)
	t.responses.AddHandler(id, c)

	select {
	case <-t.ctx.Done():
		return nil
	case result := <-c:
		r := utils.TryParseRequest[TypeToStringResponse](t.logger, result)
		return &r
	}
}

func (t *TsGo) UpdateSnapshot(tsconfig string, changes []DocumentIdentifier) *UpdateSnapshotResponse {
	t.opLock.Lock()
	defer t.opLock.Unlock()

	return t.updateSnapshotLocked(&tsconfig, changes)
}

func (t *TsGo) updateSnapshotLocked(tsconfig *string, changes []DocumentIdentifier) *UpdateSnapshotResponse {
	id := utils.GetNextStringId()

	tsGoChanges := []DocumentIdentifier{}
	tsGoCreated := []DocumentIdentifier{}

	for _, change := range changes {
		var uri string
		if change.FileName != "" {
			uri = change.FileName
		} else {
			uri = FilenameFromUri(change.URI)
		}

		if t.rootFiles[uri] {
			tsGoChanges = append(tsGoChanges, change)
		} else {
			tsGoCreated = append(tsGoCreated, change)
		}
	}

	var openProjects *[]string = nil
	if tsconfig != nil && *tsconfig != "" {
		openProjects = &[]string{*tsconfig}
	}

	request := UpdateSnapshotRequest{
		TsGoRequest: TsGoRequest{RPC: "2.0", ID: id, Method: "updateSnapshot"},
		Params: UpdateSnapshotParams{
			OpenProject:  tsconfig,
			OpenProjects: openProjects,
			FileChanges:  &APIFileChanges{Changed: tsGoChanges, Created: tsGoCreated, Deleted: []DocumentIdentifier{}},
		},
	}

	utils.WriteResponse(t.stdin, request)

	c := make(chan []byte, 1)
	t.responses.AddHandler(id, c)

	select {
	case <-t.ctx.Done():
		return nil
	case result := <-c:
		r := utils.TryParseRequest[UpdateSnapshotResponse](t.logger, result)
		t.logger.Println(string(result))
		if r.Result.Snapshot != 0 && len(r.Result.Projects) > 0 && r.Result.Projects[0].Id != "" {
			t.snapshotHandle = r.Result.Snapshot
			t.projectHandle = r.Result.Projects[0].Id

			for _, file := range r.Result.Projects[0].RootFiles {
				t.rootFiles[file] = true
			}
		}
		return &r
	}
}

// AI generated :(
// This is surely horribly fragile, but, hopefully, by the time it causes me problems, there'll be a way to
// convert a handle to a location. TsGo currently offers no such solution.
func (t *TsGo) GetNodePosition(handle NodeHandle) (pos int, end int, err error) {
	node, err := handle.ExtractNode()
	if err != nil {
		return 0, 0, err
	}

	t.opLock.Lock()

	id := utils.GetNextStringId()
	request := GetSourceFileRequest{
		TsGoRequest: TsGoRequest{RPC: "2.0", ID: id, Method: "getSourceFile"},
		Params: GetSourceFileParams{
			Snapshot: t.snapshotHandle,
			Project:  t.projectHandle,
			File:     DocumentIdentifier{URI: UriFromFilename(node.Path)},
		},
	}

	utils.WriteResponse(t.stdin, request)

	c := make(chan []byte, 1)
	t.responses.AddHandler(id, c)
	t.opLock.Unlock()

	var resultBytes []byte
	select {
	case <-t.ctx.Done():
		return 0, 0, errors.New("context done")
	case res := <-c:
		resultBytes = res
	}

	r := utils.TryParseRequest[GetSourceFileResponse](t.logger, resultBytes)
	if len(r.Result.Data) < 44 {
		return 0, 0, fmt.Errorf("invalid or missing source file data for %s", node.Path)
	}

	data := r.Result.Data
	// HEADER_OFFSET_NODES is at byte 40 (uint32)
	nodesOffset := binary.LittleEndian.Uint32(data[40:44])

	// NODE_LEN is 28 bytes. Find the specific node offset.
	nodeOffset := nodesOffset + uint32(node.Index*28)
	if int(nodeOffset+28) > len(data) {
		return 0, 0, fmt.Errorf("node index out of bounds")
	}

	// NODE_OFFSET_POS is 4, NODE_OFFSET_END is 8 (both uint32)
	pos = int(binary.LittleEndian.Uint32(data[nodeOffset+4 : nodeOffset+8]))
	end = int(binary.LittleEndian.Uint32(data[nodeOffset+8 : nodeOffset+12]))

	return pos, end, nil
}

func (t *TsGo) handleRequest(method string, contents []byte) {
	switch method {
	case "readFile":
		{
			r := utils.TryParseRequest[ReadFileRequest](t.logger, contents)
			if config.GetConfig().TsGo.ExperimentalConcurrentRequestHandling {
				go tsgoHandleReadFile(t, r)
			} else {
				tsgoHandleReadFile(t, r)
			}
		}
	case "fileExists":
		{
			r := utils.TryParseRequest[FileExistsRequest](t.logger, contents)
			if config.GetConfig().TsGo.ExperimentalConcurrentRequestHandling {
				go tsgoHandleFileExists(t, r)
			} else {
				tsgoHandleFileExists(t, r)
			}
		}
	case "getAccessibleEntries":
		{
			r := utils.TryParseRequest[GetAccessibleEntriesRequest](t.logger, contents)
			if config.GetConfig().TsGo.ExperimentalConcurrentRequestHandling {
				go tsgoHandleGetAccessibleEntries(t, r)
			} else {
				tsgoHandleGetAccessibleEntries(t, r)
			}
		}
	}
}
