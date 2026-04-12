package parser

import (
	"bufio"
	"context"
	"io"
	"log"
	"net"
	"os/exec"
	"strconv"
	"sync"
	"ts_inspector/interfaces"
	"ts_inspector/rpc"
	"ts_inspector/utils"
)

type Responses struct {
	sync.RWMutex

	response map[string]chan []byte
}

type TsGoApi struct {
	logger    *log.Logger
	nextId    int
	project   Handle
	responses *Responses
	snapshot  Handle
	state     *State

	connection *net.Conn

	ctx    context.Context
	cancel context.CancelFunc
}

type TsGoLsp struct {
	responses *Responses
	state     *State
	nextId    int

	api *TsGoApi

	stderr *io.ReadCloser
	stdin  *io.WriteCloser
	stdout *io.ReadCloser

	ctx    context.Context
	cancel context.CancelFunc
}

func StartTsGoLsp(state *State) (*TsGoLsp, error) {
	args := []string{"--lsp", "--stdio", state.GetRootPath()}
	cmd := exec.Command("/home/connor/Development/typescript-go/cmd/tsgo/tsgo", args...)

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

	lsp := &TsGoLsp{
		state:     state,
		responses: &Responses{response: make(map[string]chan []byte)},

		stderr: &stderr,
		stdin:  &stdin,
		stdout: &stdout,

		cancel: cancel,
		ctx:    ctx,
	}

	go stderrLogger(lsp)
	go runLsp(lsp)

	lsp.initialise()
	lsp.startApiSession()

	return lsp, nil
}

func (l *TsGoLsp) GetNextId() string {
	id := l.nextId
	l.nextId += 1

	return strconv.Itoa(id)
}

func (l *TsGoLsp) initialise() error {
	id := l.GetNextId()
	request := interfaces.InitializeRequest[string]{
		Request: interfaces.Request[string]{RPC: "2.0", ID: id, Method: "initialize"},
		Params: interfaces.InitializeParams{
			RootUri:      l.state.rootURI,
			Capabilities: interfaces.ClientCapabilities{},
			ProcessId:    nil,
		},
	}

	utils.WriteResponse(*l.stdin, request)

	c := make(chan []byte, 1)
	l.responses.AddHandler(id, c)

	select {
	case <-l.ctx.Done():
		return nil
	case result := <-c:
		request := interfaces.Notification{RPC: "2.0", Method: "initialized"}
		utils.WriteResponse(*l.stdin, request)
		_ = utils.TryParseRequest[InitializeAPISessionResponse](l.state.Logger, result)
		print("")
	}

	return nil
}

func (l *TsGoLsp) startApiSession() error {
	id := l.GetNextId()
	request := InitializeAPISessionRequest{
		TsGoRequest: TsGoRequest{RPC: "2.0", ID: id, Method: "custom/initializeAPISession"},
		Params:      InitializeAPISessionParams{},
	}

	utils.WriteResponse(*l.stdin, request)

	c := make(chan []byte, 1)
	l.responses.AddHandler(id, c)

	select {
	case <-l.ctx.Done():
		return nil
	case result := <-c:
		r := utils.TryParseRequest[InitializeAPISessionResponse](l.state.Logger, result)

		c, err := net.Dial("unix", r.Result.Pipe)
		if err != nil {
			return err
		}

		ctx, cancel := context.WithCancel(l.ctx)

		api := TsGoApi{
			logger: l.state.Logger,
			nextId: 0,
			state:  l.state,

			connection: &c,
			responses:  &Responses{response: make(map[string]chan []byte)},

			ctx:    ctx,
			cancel: cancel,
		}

		// api.Initialize()
		api.UpdateSnapshot(l.state.tsConfigFiles[1], nil)

		go run(&api)

		l.api = &api
	}

	return nil
}

func (l *TsGoLsp) GetApi() *TsGoApi {
	return l.api
}

// func StartTsGo(state *State) (*TsGoApi, error) {
// 	args := []string{
// 		"--api",
// 		"--async",
// 		"--callbacks",
// 		"readFile,fileExists,getAccessibleEntries",
// 		"--cwd",
// 		state.GetRootPath(),
// 	}
//
// 	cmd := exec.Command("/home/connor/Development/typescript-go/cmd/tsgo/tsgo", args...)
//
// 	if err := cmd.Start(); err != nil {
// 		return nil, err
// 	}
//
// 	ctx, cancel := context.WithCancel(context.Background())
//
// 	tsgo := &TsGoApi{
// 		logger:    state.Logger,
// 		nextId:    0,
// 		state:     state,
// 		responses: &Responses{response: make(map[string]chan []byte)},
//
// 		cancel: cancel,
// 		ctx:    ctx,
// 	}
//
// 	go run(tsgo)
// 	go stderrLogger(tsgo)
//
// 	return tsgo, nil
// }

func runLsp(lsp *TsGoLsp) {
	scanner := bufio.NewScanner(*lsp.stdout)

	scanner.Split(rpc.Split)
	big := 1024 * 1024 // 1 mb
	buf := make([]byte, big)
	scanner.Buffer(buf, big)

	for scanner.Scan() {
		msg := scanner.Bytes()
		lsp.state.Logger.Println(string(msg))
		_, contents, err := rpc.DecodeMessage(msg)
		if err != nil {
			continue
		}

		r := utils.TryParseRequest[TsGoRequest](lsp.state.Logger, contents)
		if r.Method == "" {
			lsp.responses.GetHandler(r.ID) <- contents
		} else {
			lsp.handleRequest(r.Method, contents)
		}
	}
}

func run(tsgo *TsGoApi) {
	scanner := bufio.NewScanner(*tsgo.connection)

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
			handler := tsgo.responses.GetHandler(r.ID)
			if handler != nil {
				handler <- contents
			}
		} else {
			tsgo.handleRequest(r.Method, contents)
		}
	}
}

func stderrLogger(lsp *TsGoLsp) {
	scanner := bufio.NewScanner(*lsp.stderr)

	big := 1024 * 1024 // 1 mb
	buf := make([]byte, big)
	scanner.Buffer(buf, big)

	for scanner.Scan() {
		msg := scanner.Bytes()
		lsp.state.Logger.Println(string(msg))
	}
}

func (r *Responses) AddHandler(id string, channel chan []byte) {
	r.RLock()
	r.response[id] = channel
	r.RUnlock()
}

func (r *Responses) GetHandler(id string) chan []byte {
	r.RLock()
	c := r.response[id]
	r.RUnlock()

	return c
}

func (t *TsGoApi) GetNextId() string {
	id := t.nextId
	t.nextId += 1

	return strconv.Itoa(id)
}

func (t *TsGoApi) GetSemanticDiagnostics(uri string) *DiagnosticResponse {
	t.UpdateSnapshot("", &APIFileChanges{Changed: []DocumentIdentifier{{URI: uri}}})

	id := t.GetNextId()
	request := GetDiagnosticsRequest{
		TsGoRequest: TsGoRequest{RPC: "2.0", ID: id, Method: "getSemanticDiagnostics"},
		Params: GetDiagnosticsParams{
			Snapshot: t.snapshot,
			Project:  t.project,
			File:     &DocumentIdentifier{URI: uri},
		},
	}

	utils.WriteResponse(*t.connection, request)

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

func (t *TsGoApi) GetSymbolAtPosition(uri string, offset uint32) *SymbolResponse {
	t.UpdateSnapshot("", &APIFileChanges{Changed: []DocumentIdentifier{{URI: uri}}})

	id := t.GetNextId()
	request := GetSymbolAtPositionRequest{
		TsGoRequest: TsGoRequest{RPC: "2.0", ID: id, Method: "getSymbolAtPosition"},
		Params: GetSymbolAtPositionParams{
			File:     DocumentIdentifier{URI: uri},
			Position: offset,
			Project:  t.project,
			Snapshot: t.snapshot,
		},
	}

	utils.WriteResponse(*t.connection, request)

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

func (t *TsGoApi) GetTypeOfSymbol(symbol Handle) *TypeResponse {
	id := t.GetNextId()
	request := GetTypeOfSymbolRequest{
		TsGoRequest: TsGoRequest{RPC: "2.0", ID: id, Method: "getTypeOfSymbol"},
		Params: GetTypeOfSymbolParams{
			Project:  t.project,
			Snapshot: t.snapshot,
			Symbol:   symbol,
		},
	}

	utils.WriteResponse(*t.connection, request)

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

func (t *TsGoApi) Initialize() *InitializeResponse {
	id := t.GetNextId()
	request := TsGoRequest{RPC: "2.0", ID: id, Method: "initialize"}

	utils.WriteResponse(*t.connection, request)

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

func (t *TsGoApi) TypeToString(ttype Handle) *TypeToStringResponse {
	id := t.GetNextId()
	request := TypeToTypeNodeRequest{
		TsGoRequest: TsGoRequest{RPC: "2.0", ID: id, Method: "typeToString"},
		Params: TypeToTypeNodeParams{
			Project:  t.project,
			Snapshot: t.snapshot,
			Type:     ttype,
		},
	}

	utils.WriteResponse(*t.connection, request)

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

func (t *TsGoApi) UpdateSnapshot(tsconfig string, changes *APIFileChanges) *UpdateSnapshotResponse {
	id := t.GetNextId()
	request := UpdateSnapshotRequest{
		TsGoRequest: TsGoRequest{RPC: "2.0", ID: id, Method: "updateSnapshot"},
		Params:      UpdateSnapshotParams{OpenProject: tsconfig, FileChanges: changes},
	}
	utils.WriteResponse(*t.connection, request)

	c := make(chan []byte, 1)
	t.responses.AddHandler(id, c)

	select {
	case <-t.ctx.Done():
		return nil
	case result := <-c:
		r := utils.TryParseRequest[UpdateSnapshotResponse](t.logger, result)
		t.logger.Println(string(result))

		if len(r.Result.Projects) > 0 && r.Result.Projects[0].Id != "" {
			t.project = r.Result.Projects[0].Id
		}

		if r.Result.Snapshot != "" {
			t.snapshot = r.Result.Snapshot
		}
		return &r
	}
}

func (t *TsGoApi) handleRequest(method string, contents []byte) {
	switch method {
	case "readFile":
		{
			r := utils.TryParseRequest[ReadFileRequest](t.logger, contents)
			go tsgoHandleReadFile(t, r)
		}
	case "fileExists":
		{
			r := utils.TryParseRequest[FileExistsRequest](t.logger, contents)
			go tsgoHandleFileExists(t, r)
		}
	case "getAccessibleEntries":
		{
			r := utils.TryParseRequest[GetAccessibleEntriesRequest](t.logger, contents)
			go tsgoHandleGetAccessibleEntries(t, r)
		}
	}
}

func (t *TsGoLsp) handleRequest(method string, contents []byte) {
	r := utils.TryParseRequest[TsGoRequest](t.state.Logger, contents)
	request := TsGoResponse{RPC: "2.0", ID: r.ID}

	utils.WriteResponse(*t.stdin, request)
}
