package printer

type jsonPrinter struct {}

func NewJSONPrinter() Printer {
	return &jsonPrinter{}
}

func (p *jsonPrinter) Print(v interface{}) error {
	return nil
}
