package alfredo

import "testing"

func TestWebHookStruct_SendMsg(t *testing.T) {
	type fields struct {
		WebHookURL string
	}
	type args struct {
		msg string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "base test",
			fields: fields{
				WebHookURL: GetFirstLineFromFile("./webhook.url"),
			},
			args: args{
				msg: "test run",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wh := WebHookStruct{
				WebHookURL: tt.fields.WebHookURL,
			}
			if err := wh.SendMsg(tt.args.msg); (err != nil) != tt.wantErr {
				t.Errorf("WebHookStruct.SendMsg() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
