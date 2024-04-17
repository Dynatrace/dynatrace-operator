package dttoken

type Token struct {
	prefix  string
	public  string
	private string
}

func New(prefix string) *Token {

	return &Token{prefix: prefix}
}

func (t *Token) generate() string {
	return ""
}
