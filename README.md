#Pipeline operator

This is a kubernetes operator designed to add the "pipeline" kind that runs a set of pods in sequence. Each pod can use a different image and entrypoint, and the stdout from each is passed to the next by means of an environment variable. This is either user defined or takes the name of the pipeline

An example of the configuration needed is in the pipeline-cr.yaml file
