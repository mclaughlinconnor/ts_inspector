package search

// I think go-faiss bindings have a bug that lets the underlying pointer in an IDMap index can seg fault
// Do the FAISS stuff myself

/*
#cgo LDFLAGS: -lfaiss_c

#include <faiss/c_api/IndexFlat_c.h>
#include <faiss/c_api/MetaIndexes_c.h>
*/
import "C"

import (
	"log"
	"unsafe"
)

var dimension = 384

var index *C.FaissIndex

type Result struct {
	Distance float32
	Id       int64
}

type Vector struct {
	Id     int64
	Vector []float32
}

func AddToFAISS(vectors []Vector) {
	nVectors := len(vectors)
	combinedVectors := make([]float32, nVectors*dimension)
	ids := make([]int64, nVectors)

	for i, v := range vectors {
		for j := range dimension {
			combinedVectors[(dimension*i)+j] = v.Vector[j]
		}
		ids[i] = v.Id
	}

	addWithIds(index, nVectors, combinedVectors, ids)
}

func SearchFAISS(queryVector []float32, resultsCount int64) ([]Result, error) {
	distances, labels := indexSearch(queryVector, resultsCount)

	results := make([]Result, resultsCount)
	for i := range resultsCount {
		results[i] = Result{distances[i], labels[i]}
	}

	return results, nil
}

func addWithIds(index *C.FaissIndex, n int, vectors []float32, ids []int64) {
	ret := C.faiss_Index_add_with_ids(
		index,
		C.idx_t(n),
		(*C.float)(unsafe.Pointer(&vectors[0])),
		(*C.idx_t)(unsafe.Pointer(&ids[0])),
	)

	if ret != 0 {
		log.Fatal("faiss_Index_add_with_ids failed")
	}
}

func indexSearch(queryVector []float32, resultsCount int64) ([]float32, []int64) {
	distances := make([]float32, resultsCount)
	labels := make([]int64, resultsCount)

	ret := C.faiss_Index_search(
		index,
		C.idx_t(1),
		(*C.float)(unsafe.Pointer(&queryVector[0])),
		C.idx_t(resultsCount),
		(*C.float)(unsafe.Pointer(&distances[0])),
		(*C.idx_t)(unsafe.Pointer(&labels[0])),
	)

	if ret != 0 {
		log.Fatalf("faiss_Index_search failed")
	}

	return distances, labels
}

func initFAISS() {
	var flatIndex *C.FaissIndex = newIndexFlatL2()
	var idmapIndex *C.FaissIndex = newIndexIdMap(flatIndex)

	index = idmapIndex
}

func newIndexFlatL2() *C.FaissIndex {
	var flatIndex *C.FaissIndex

	ret := C.faiss_IndexFlatL2_new_with(
		(**C.FaissIndexFlatL2)(unsafe.Pointer(&flatIndex)),
		C.idx_t(dimension),
	)
	if ret != 0 {
		log.Fatal("faiss_IndexFlatL2_new_with failed")
	}

	return flatIndex
}

func newIndexIdMap(flatIndex *C.FaissIndex) *C.FaissIndex {
	var idMapIndex *C.FaissIndex

	ret := C.faiss_IndexIDMap_new(
		(**C.FaissIndexIDMap)(unsafe.Pointer(&idMapIndex)),
		flatIndex,
	)
	if ret != 0 {
		log.Fatal("faiss_IndexIDMap_new failed")
	}

	return idMapIndex
}
