package printer

type Printer interface {
	Print(v interface{}) error
}
