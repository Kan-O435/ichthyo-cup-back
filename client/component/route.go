export fung Route() {
	return (
		r := router.New()
		r.Handle("/", &HomePage{})
		r.Handle("/home", &HomePage{})
	)
}