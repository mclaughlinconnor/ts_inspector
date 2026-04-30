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
	project         Handle
	requestHandlers map[string]func(request TsGoRequest) any
	responses       *Responses
	snapshot        Handle
	state           *State
	virtualFiles    map[string]string

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

	tsgo := &TsGo{
		logger:    state.Logger,
		nextId:    0,
		state:     state,
		responses: &Responses{response: make(map[string]chan []byte)},

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

func (t *TsGo) GetTypeOfSymbol(symbol Handle) *TypeResponse {
	id := t.GetNextId()
	request := GetTypeOfSymbolRequest{
		TsGoRequest: TsGoRequest{RPC: "2.0", ID: id, Method: "getTypeOfSymbol"},
		Params: GetTypeOfSymbolParams{
			Project:  t.project,
			Snapshot: t.snapshot,
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

func (t *TsGo) TypeToString(ttype Handle) *TypeToStringResponse {
	id := t.GetNextId()
	request := TypeToTypeNodeRequest{
		TsGoRequest: TsGoRequest{RPC: "2.0", ID: id, Method: "typeToString"},
		Params: TypeToTypeNodeParams{
			Project:  t.project,
			Snapshot: t.snapshot,
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

func (t *TsGo) UpdateSnapshot(tsconfig string, changes *APIFileChanges) *UpdateSnapshotResponse {
	id := t.GetNextId()
	request := UpdateSnapshotRequest{
		TsGoRequest: TsGoRequest{RPC: "2.0", ID: id, Method: "updateSnapshot"},
		Params:      UpdateSnapshotParams{OpenProject: tsconfig, FileChanges: changes},
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
		if r.Result.Snapshot != "" && len(r.Result.Projects) > 0 && r.Result.Projects[0].Id != "" {
			t.snapshot = r.Result.Snapshot
			t.project = r.Result.Projects[0].Id
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
