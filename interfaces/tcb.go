package interfaces

const TCB_FILENAME_SUFFIX = ".ts_inspector-tcb.ts"

type TcbParams struct {
	Uri string `json:"uri"`
}

type TcbRequest struct {
	RequestMessage

	Params TcbParams `json:"params"`
}

type TcbRequestResult = string

type TcbRequestResponse struct {
	ResponseMessage

	Result TcbRequestResult `json:"result"`
}
