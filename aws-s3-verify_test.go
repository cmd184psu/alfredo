package alfredo

import (
	"github.com/aws/aws-sdk-go/service/s3"
)

type fakeIter struct {
	objs []*s3.Object
	idx  int
}

func (f *fakeIter) Next() (*s3.Object, error) {
	if f.idx >= len(f.objs) {
		return nil, nil
	}
	o := f.objs[f.idx]
	f.idx++
	return o, nil
}

// func TestVerifyDetectsMissing(t *testing.T) {
// 	src := &fakeIter{
// 		objs: []*s3.Object{
// 			{Key: str("a"), Size: int64p(1)},
// 		},
// 	}
// 	dst := &fakeIter{}

// 	var buf bytes.Buffer
// 	enc := json.NewEncoder(&buf)
// 	enc.Encode(Diff{Type: Missing, Key: "a", SourceSize: 1})
// }

func str(s string) *string  { return &s }
func int64p(i int64) *int64 { return &i }
