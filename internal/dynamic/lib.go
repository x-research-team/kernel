package dynamic

// Lambda Функция выполнения
type Lambda interface{}

// List Коллекция объектов
func List(s ...interface{}) []interface{} {
	return s
}
