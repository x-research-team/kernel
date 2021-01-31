package generator

// Strings Асинхронный генератор строк
func Strings(list []string, abort <-chan struct{}) <-chan string {
	ch := make(chan string)
	go func(list []string) {
		defer close(ch)
		for _, s := range list {
			select {
			case ch <- s:
			case <-abort: // receive on closed channel can proceed immediately
				return
			default:
				continue
			}
		}
	}(list)
	return ch
}

// Errors Асинхронный генератор ошибок
func Errors(list []error, abort <-chan struct{}) <-chan error {
	ch := make(chan error)
	go func(list []error) {
		defer close(ch)
		for _, s := range list {
			select {
			case ch <- s:
			case <-abort: // receive on closed channel can proceed immediately
				return
			default:
				continue
			}
		}
	}(list)
	return ch
}
