package angular_content

//#include "parser.h"
//TSLanguage *tree_sitter_angular_content();
import "C"
import (
	"unsafe"

	sitter "github.com/smacker/go-tree-sitter"
)

func GetLanguage() *sitter.Language {
	ptr := unsafe.Pointer(C.tree_sitter_angular_content())
	return sitter.NewLanguage(ptr)
}
