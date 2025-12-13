package log

// Config 日志配置结构体
type Config struct {
	// Level 设置日志的最低记录级别
	Level Level
	// StdoutEnable 是否输出到标准输出，设置为true时，日志会输出到标准输出，和文件可以同时存在
	StdoutEnable bool
	// FileOutEnable 是否输出到文件，设置为true时，日志会输出到文件，和标准输出可以同时存在
	FileOutEnable bool
	// OutputDir 指定日志文件输出目录，为空则输出到标准输出
	OutputDir string
	// Filename 指定日志文件名，默认为"app.log"
	Filename string
	// ErrorSperate 是否将错误日志与正常日志分离开来，设置为true时，错误日志会输出到单独的文件
	ErrorSperate bool
	// ErrorFilename 指定错误日志文件名，默认为Filename+"_error"
	ErrorFilename string
	// MaxSize 单个日志文件的最大大小（MB），默认为100MB
	MaxSize int
	// MaxAge 日志文件的最大保留天数，默认为7天
	MaxAge int
	// ByDate 是否按日期分文件，设置为true时，日志文件名会包含日期
	ByDate bool
	// Development 是否为开发模式，开发模式下日志更易读
	Development bool
	// Encoding 日志编码方式，json或console
	Encoding string
	// BufferSize 设置日志缓冲区大小（字节），0表示使用默认值
	BufferSize int
	// FlushInterval 设置自动刷新间隔（秒），0表示不自动刷新
	FlushInterval int
	// FlushOnWrite 设置是否在每次写入后立即刷新，适用于关键日志
	FlushOnWrite bool
}
