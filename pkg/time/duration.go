package time
import xtime "time"

type Duration xtime.Duration
// *用于从文本（如 1s、500ms）解析为 time.Duration 类型
func (d *Duration) UnmarshalText(text []byte) error {
	tmp, err := xtime.ParseDuration(string(text))
	if err == nil {
		*d = Duration(tmp)
	}
	return err
}
