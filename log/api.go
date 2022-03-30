package log

func Error(err error, fields ...interface{}) {
	if err == nil {
		return
	}
	e := logger.Err(err)
	if len(fields) > 0 {
		e = e.Fields(fields)
	}

	e.Send()
}

func Debug(msg string, fields ...interface{}) {
	e := logger.Debug()
	if len(fields) > 0 {
		e = e.Fields(fields)
	}

	e.Msg(msg)
}

func Info(msg string, fields ...interface{}) {
	e := logger.Info()
	if len(fields) > 0 {
		e = e.Fields(fields)
	}

	e.Msg(msg)
}

func Warn(msg string, fields ...interface{}) {
	e := logger.Warn()
	if len(fields) > 0 {
		e = e.Fields(fields)
	}

	e.Msg(msg)
}

func Fatal(i interface{}, fields ...interface{}) {
	if i == nil {
		return
	}
	e := logger.Fatal()
	if len(fields) > 0 {
		e = e.Fields(fields)
	}

	switch o := i.(type) {
	case error:
		e.Err(o).Send()

		return
	case string:
		e.Msg(o)

		return
	default:
		e.Send()

		return
	}
}

func Panic(i interface{}, fields ...interface{}) {
	if i == nil {
		return
	}
	e := logger.Panic()
	if len(fields) > 0 {
		e = e.Fields(fields)
	}

	switch o := i.(type) {
	case error:
		e.Err(o).Send()

		return
	case string:
		e.Msg(o)

		return
	default:
		e.Send()

		return
	}
}

func Level() string {
	return logger.GetLevel().String()
}
