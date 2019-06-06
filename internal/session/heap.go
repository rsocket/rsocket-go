package session

type sHeap []*Session

func (h sHeap) Len() int {
	return len(h)
}

func (h sHeap) Less(i, j int) bool {
	return h[i].deadline.Before(h[j].deadline)
}

func (h sHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
	h[i].index, h[j].index = i, j
}

func (h *sHeap) Push(x interface{}) {
	session := x.(*Session)
	session.index = len(*h)
	*h = append(*h, session)
}

func (h *sHeap) Pop() interface{} {
	old := *h
	n := len(old)
	x := old[n-1]
	old[n-1] = nil
	*h = old[0 : n-1]
	x.index = -1
	return x
}
