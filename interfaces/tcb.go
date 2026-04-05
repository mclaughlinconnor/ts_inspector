package interfaces

const TCB_FILENAME_SUFFIX = ".ts_inspector-tcb.ts"

type TcbParams struct {
	Uri string `json:"uri"`
}

type TcbRequest struct {
	Request

	Params TcbParams `json:"params"`
}

type TcbRequestResult = string

type TcbRequestResponse struct {
	Response

	Result TcbRequestResult `json:"result"`
}
