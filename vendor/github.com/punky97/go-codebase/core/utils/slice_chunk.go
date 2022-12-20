package utils

//slice_int64 chunk slice by lim
func SliceInt64(input []int64, lim int) [][]int64 {
	var chunk []int64
	chunks := make([][]int64, 0, len(input)/lim+1)
	for len(input) >= lim {
		chunk, input = input[:lim], input[lim:]
		chunks = append(chunks, chunk)
	}
	if len(input) > 0 {
		chunks = append(chunks, input[:len(input)])
	}
	return chunks
}
