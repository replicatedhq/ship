resource "local_file" "foo" {
  content     = "foo!"
  filename = "/tmp/foo.bar"
}
