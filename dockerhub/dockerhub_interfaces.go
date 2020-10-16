package dockerhub

// DockerHub registry API basic operations
type DockerHub interface {
	Tags(repostiory string) ([]string, error)
}
