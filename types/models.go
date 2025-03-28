package types

type Parameter struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

type Function struct {
	Name string `json:"name"`
	Parameters []Parameter `json:"parameters"`
	ReturnTypes []string `json:"return-types"`
	Body string `json:"body"`
}

type Method struct {
	Reciever string `json:"reciever"`
	Function
}

type Field struct {
	Name string `json:"name"`
	Type string `json:"type"`
}

type Class struct {
	Name string `json:"name"`
	Fields []Field `json:"fields"`
	Methods []Method `json:"methods"`
}

type Interface struct {
	Name string `json:"name"`
	Methods []Function `json:"methods"`
}
