config: {
  debugPort: 8888
  file: {
    match: "go.mod" 
    required: true
  }
  dependsOn: {
    host: "localhsot:2222"
  }
}
