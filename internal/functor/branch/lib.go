package branch

import (
	"log"

	"github.com/x-research-team/kernel/internal/functor"
)

// Pipe (f: func (...any): []any) Выполнить команду в ветке исполнения
func Pipe(pipes ...interface{}) func(*functor.Functor) *functor.Functor {
	return func(f *functor.Functor) *functor.Functor {
		var err error
		for _, p := range pipes {
			f, err = f.Pipe(p)
			if err != nil {
				log.Printf("%v\n", err)
				break
			}
		}
		return f
	}
}
