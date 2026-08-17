package parser

import (
	"bufio"
	"context"
	"io"
	"log"
	"os/exec"
	"strconv"
	"sync"
	"ts_inspector/rpc"
	"ts_inspector/utils"
)

type Responses struct {
	sync.RWMutex

	response map[string]chan []byte
}

type TsGo struct {
	logger          *log.Logger
	nextId          int
	projectHandle   ProjectID
	requestHandlers map[string]func(request TsGoRequest) any
	responses       *Responses
	rootFiles       map[string]bool
	snapshotHandle  SnapshotID
	state           *State

	stderr *io.ReadCloser
	stdin  *io.WriteCloser
	stdout *io.ReadCloser

	ctx    context.Context
	cancel context.CancelFunc
}

func StartTsGo(state *State) (*TsGo, error) {
	args := []string{
		"--api",
		"--async",
		"--callbacks",
		"readFile,fileExists,getAccessibleEntries",
		"--cwd",
		state.GetRootPath(),
	}

	cmd := exec.Command("/home/connor/.local/share/nvim/mason/bin/tsgo", args...)

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

	ctx, cancel := context.WithCancel(context.Background())

	tsgo := &TsGo{
		logger:    state.Logger,
		nextId:    0,
		responses: &Responses{response: make(map[string]chan []byte)},
		rootFiles: make(map[string]bool),
		state:     state,

		stderr: &stderr,
		stdin:  &stdin,
		stdout: &stdout,

		cancel: cancel,
		ctx:    ctx,
	}

	go run(tsgo)
	go stderrLogger(tsgo)

	return tsgo, nil
}

func run(tsgo *TsGo) {
	scanner := bufio.NewScanner(*tsgo.stdout)

	scanner.Split(rpc.Split)
	big := 1024 * 1024 // 1 mb
	buf := make([]byte, big)
	scanner.Buffer(buf, big)

	for scanner.Scan() {
		msg := scanner.Bytes()
		tsgo.logger.Println(string(msg))
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

func (t *TsGo) GetNextId() string {
	id := t.nextId
	t.nextId += 1

	return strconv.Itoa(id)
}

func (t *TsGo) GetSemanticDiagnostics(uri string) *DiagnosticResponse {
	t.UpdateSnapshot("", []DocumentIdentifier{{URI: uri}})

	id := t.GetNextId()
	request := GetDiagnosticsRequest{
		TsGoRequest: TsGoRequest{RPC: "2.0", ID: id, Method: "getSemanticDiagnostics"},
		Params: GetDiagnosticsParams{
			Snapshot: t.snapshotHandle,
			Project:  t.projectHandle,
			File:     &DocumentIdentifier{URI: uri},
		},
	}

	utils.WriteResponse(*t.stdin, request)

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

func (t *TsGo) GetSymbolAtPosition(uri string, offset uint32) *SymbolResponse {
	t.UpdateSnapshot("", []DocumentIdentifier{{URI: uri}})

	id := t.GetNextId()
	request := GetSymbolAtPositionRequest{
		TsGoRequest: TsGoRequest{RPC: "2.0", ID: id, Method: "getSymbolAtPosition"},
		Params: GetSymbolAtPositionParams{
			File:     DocumentIdentifier{URI: uri},
			Position: offset,
			Project:  t.projectHandle,
			Snapshot: t.snapshotHandle,
		},
	}

	utils.WriteResponse(*t.stdin, request)

	c := make(chan []byte, 1)
	t.responses.AddHandler(id, c)

	select {
	case <-t.ctx.Done():
		return nil
	case result := <-c:
		r := utils.TryParseRequest[SymbolResponse](t.logger, result)
		return &r
	}
}

func (t *TsGo) GetTypeOfSymbol(symbol SymbolID) *TypeResponse {
	id := t.GetNextId()
	request := GetTypeOfSymbolRequest{
		TsGoRequest: TsGoRequest{RPC: "2.0", ID: id, Method: "getTypeOfSymbol"},
		Params: GetTypeOfSymbolParams{
			Project:  t.projectHandle,
			Snapshot: t.snapshotHandle,
			Symbol:   symbol,
		},
	}

	utils.WriteResponse(*t.stdin, request)

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

func (t *TsGo) GetTypeAtPosition(uri string, offset uint32) *TypeResponse {
	documentIdentifier := DocumentIdentifier{URI: uri}
	t.UpdateSnapshot("", []DocumentIdentifier{documentIdentifier})

	id := t.GetNextId()
	request := GetTypeAtPositionParamsRequest{
		TsGoRequest: TsGoRequest{RPC: "2.0", ID: id, Method: "getTypeAtPosition"},
		Params: GetTypeAtPositionParams{
			Project:  t.projectHandle,
			Snapshot: t.snapshotHandle,
			File:     documentIdentifier,
			Position: offset,
		},
	}

	utils.WriteResponse(*t.stdin, request)

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
	id := t.GetNextId()
	request := TsGoRequest{RPC: "2.0", ID: id, Method: "initialize"}

	utils.WriteResponse(*t.stdin, request)

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
	id := t.GetNextId()
	request := TypeToTypeNodeRequest{
		TsGoRequest: TsGoRequest{RPC: "2.0", ID: id, Method: "typeToString"},
		Params: TypeToTypeNodeParams{
			Flags:    TypeFormatFlagsUseAliasDefinedOutsideCurrentScope | TypeFormatFlagsUseInstantiationExpressions,
			Project:  t.projectHandle,
			Snapshot: t.snapshotHandle,
			Type:     ttype,
		},
	}

	utils.WriteResponse(*t.stdin, request)

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
	id := t.GetNextId()

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
			OpenProjects: openProjects,
			FileChanges:  &APIFileChanges{Changed: tsGoChanges, Created: tsGoCreated, Deleted: []DocumentIdentifier{}},
		},
	}

	utils.WriteResponse(*t.stdin, request)

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

func (t *TsGo) handleRequest(method string, contents []byte) {
	switch method {
	case "readFile":
		{
			r := utils.TryParseRequest[ReadFileRequest](t.logger, contents)
			tsgoHandleReadFile(t, r)
		}
	case "fileExists":
		{
			r := utils.TryParseRequest[FileExistsRequest](t.logger, contents)
			tsgoHandleFileExists(t, r)
		}
	case "getAccessibleEntries":
		{
			r := utils.TryParseRequest[GetAccessibleEntriesRequest](t.logger, contents)
			tsgoHandleGetAccessibleEntries(t, r)
		}
	}
}
