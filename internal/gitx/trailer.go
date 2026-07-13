package gitx

func (r Repo) AddTrailer(file, key, value string) error {
	_, err := r.Output("interpret-trailers", "--if-exists", "addIfDifferent",
		"--trailer", key+": "+value, "--in-place", file)
	return err
}
