package audit

type Meta map[string]interface{}

type FuncMetaTypeConv func(val interface{}) (newVal interface{}, converted bool)

type Record struct {
	APIPath   string
	Event     string
	Status    string
	UserID    string
	SessionID string
	Client    string
	IPAddress string
	Meta      Meta
	metaConv  []FuncMetaTypeConv
}

func (rec *Record) Success() {
	rec.Status = Success
}

func (rec *Record) Fail() {
	rec.Status = Fail
}

func (rec *Record) AddMeta(name string, val interface{}) {
	if rec.Meta == nil {
		rec.Meta = Meta{}
	}

	var converted bool
	for _, conv := range rec.metaConv {
		val, converted = conv(val)
		if converted {
			break
		}
	}

	rec.Meta[name] = val
}

func (rec *Record) AddMetaTypeConverter(f FuncMetaTypeConv) {
	rec.metaConv = append(rec.metaConv, f)
}
