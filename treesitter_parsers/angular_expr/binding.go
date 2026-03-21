package angular_expr

//#include "parser.h"
//TSLanguage *tree_sitter_angular_expr();
import "C"
import (
	"unsafe"

	sitter "github.com/smacker/go-tree-sitter"
)

func GetLanguage() *sitter.Language {
	ptr := unsafe.Pointer(C.tree_sitter_angular_expr())
	return sitter.NewLanguage(ptr)
}
