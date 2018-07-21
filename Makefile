deploy-bootstrap-example:
	GOOS=linux go1.11beta2 build  -o dist/bootstrap ./example/bootstrap
	scp dist/bootstrap lcm@schutzenberger.generictestdomain.net:~/bootstrap
	ssh lcm@schutzenberger.generictestdomain.net "/home/lcm/bootstrap"