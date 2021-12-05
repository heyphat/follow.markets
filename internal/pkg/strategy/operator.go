package strategy

type Operator string

const (
	Less      Operator = "less"
	More      Operator = "more"
	Equal     Operator = "equal"
	LessEqual Operator = "less_equal"
	MoreEqual Operator = "more_equal"

	Or  Operator = "or"
	And Operator = "and"
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
	default:
		return "unknow"
	}
}

func (o *Operator) copy() *Operator {
	var no Operator
	no = *o
	return &no
}
