package stream

import "sync"

func FanIn[T any](streams ...<-chan T) <-chan T {
	out := make(chan T, len(streams))

	var wg sync.WaitGroup
	receive := func(c <-chan T) {
		for v := range c {
			out <- v
		}
		wg.Done()
	}

	wg.Add(len(streams))
	for _, stream := range streams {
		go receive(stream)
	}

	go func() {
		wg.Wait()
		close(out)
	}()

	return out
}
