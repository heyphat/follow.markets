package strategy

type Operator string

const (
	Less      Operator = "LESS"
	More      Operator = "MORE"
	Equal     Operator = "EQUAL"
	LessEqual Operator = "LESS_EQUAL"
	MoreEqual Operator = "MORE_EQUAL"
	NotEqual  Operator = "NOT_EQUAL"

	Or  Operator = "OR"
	And Operator = "AND"
)

func (o *Operator) toString() string {
	switch *o {
	case Less:
		return "<"
	case More:
		return ">"
	case Equal:
		return "="
	case LessEqual:
		return "<="
	case MoreEqual:
		return ">="
	case NotEqual:
		return "<>"
	default:
		return "UNKNOWN"
	}
}

func (o *Operator) copy() *Operator {
	var no Operator
	no = *o
	return &no
}
