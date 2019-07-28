package printer

type tablePrinter struct {}

func NewTablePrinter() Printer {
	return &tablePrinter{}
}

func (p *tablePrinter) Print(v interface{}) error {
	return nil
}
