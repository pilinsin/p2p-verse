package __

import(
	"time"
	proto "google.golang.org/protobuf/proto"
)

type TimeParams struct{
	Name string
	Begin time.Time
	End time.Time
	Eps time.Duration
	Cool time.Duration
	N int
}
func (tp *TimeParams) Marshal() ([]byte, error){
	bg, _ := tp.Begin.MarshalBinary()
	ed, _ := tp.End.MarshalBinary()
	ep := int64(tp.Eps)
	cl := int64(tp.Cool)
	tm := &TimeMessage{
		Name: tp.Name,
		Begin: bg,
		End: ed,
		Eps: ep,
		Cool: cl,
		N: int32(tp.N),
	}
	return proto.Marshal(tm)
}
func (tp *TimeParams) Unmarshal(m []byte) error{
	tm := &TimeMessage{}
	if err := proto.Unmarshal(m, tm); err != nil{return err}

	bg := time.Time{}
	if err := bg.UnmarshalBinary(tm.GetBegin()); err != nil{return err}
	ed := time.Time{}
	if err := ed.UnmarshalBinary(tm.GetEnd()); err != nil{return err}
	ep := time.Duration(tm.GetEps())
	cl := time.Duration(tm.GetCool())

	tp.Name = tm.GetName()
	tp.Begin = bg
	tp.End = ed
	tp.Eps = ep
	tp.Cool = cl
	tp.N = int(tm.GetN())
	return nil
}
