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
