package utils

type BufferChan struct {
	bufferSize int
	buff []string

	closed chan struct{}
}

func NewBufferChan(size int) *BufferChan {
	return &BufferChan{
		bufferSize: size,
		buff: make([]string, 0, size),
		closed: make(chan struct{}),
	}
}

func (b *BufferChan) BufferString(in chan string, out chan string) {
	defer close(out)

	var inflight = in

	for delivery := range in {
		b.buff = append(b.buff, delivery)

		for len(b.buff) > 0 {
			select {
			case <-b.closed:
				// closed before drained, drop in-flight
				return

			case delivery, consuming := <-inflight:
				if consuming {
					b.buff = append(b.buff, delivery)
				} else {
					inflight = nil
				}

			case out <- b.buff[0]:
				b.buff = b.buff[1:]
			}
		}
	}
}

func (b *BufferChan) Len() int {
	return len(b.buff)
}

func (b *BufferChan) Close() {
	close(b.closed)
}
