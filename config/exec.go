package config

type ExecProcess struct {
	Args []string
}

func (ep *ExecProcess) Run() error {
	args := ep.Args
	if args == nil || len(args) == 0 {
		args = []string{} // TODO: 実行スべきコマンド名を特定する
	}
	return execCmd(args)
}
