load('ext://restart_process', 'docker_build_with_restart')

def kubebuilder(DOMAIN, GROUP, VERSION, KIND, IMG='controller:latest', CONTROLLERGEN='rbac:roleName=manager-role crd webhook paths="./..." output:crd:artifacts:config=config/crd/bases;', DISABLE_SECURITY_CONTEXT=True):

    DOCKERFILE = '''FROM golang:alpine
    WORKDIR /
    COPY ./tilt_bin/manager /
    CMD ["/manager"]
    '''

    def yaml():
        data = local('cd config/manager; ./bin/kustomize edit set image controller=' + IMG + '; cd ../..; ./bin/kustomize build config/default')
        if DISABLE_SECURITY_CONTEXT:
            decoded = decode_yaml_stream(data)
            if decoded:
                for d in decoded:
                    # Live update conflicts with SecurityContext, until a better solution, just remove it
                    if d["kind"] == "Deployment":
                        if "securityContext" in d['spec']['template']['spec']:
                            d['spec']['template']['spec'].pop('securityContext')
                        for c in d['spec']['template']['spec']['containers']:
                            if "securityContext" in c:
                                c.pop('securityContext')

            return encode_yaml_stream(decoded)
        return data

    def manifests():
        return './bin/controller-gen ' + CONTROLLERGEN

    def generate():
        return './bin/controller-gen object:headerFile="hack/boilerplate.go.txt" paths="./...";'

    def vetfmt():
        return 'go vet ./...; go fmt ./...'

    # build to tilt_bin beause kubebuilder has a dockerignore for bin/
    def binary():
        return 'CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GO111MODULE=on go build -o tilt_bin/manager cmd/main.go'

    installed = local("which kubebuilder")
    print("kubebuilder is present:", installed)

    DIRNAME = os.path.basename(os. getcwd())
    # if kubebuilder
    if not os.path.exists('go.mod'):
        local("go mod init %s" % DIRNAME)

    if not os.path.exists('PROJECT'):
        local("kubebuilder init --domain %s" % DOMAIN)

    if not os.path.exists('api'):
        local("kubebuilder create api --resource --controller --group %s --version %s --kind %s" % (GROUP, VERSION, KIND))

    local(manifests())
    local(generate())

    local_resource('CRD', './bin/kustomize build config/crd | kubectl apply -f -', deps=["api"])

    k8s_yaml(yaml())

    COMPILE_DEPS = [
        'internal/controller',
        'cmd/main.go',
        'api', # Still include api dir as compilation might depend on API changes
        'go.mod',
        'go.sum'
    ]
    GENERATE_DEPS = [
        'api', # Trigger generate on API changes
        'hack/boilerplate.go.txt' # Trigger generate on boilerplate changes
    ]
    local_resource(
        'Generate Code',
        generate(),
        deps=GENERATE_DEPS,
        resource_deps=[], # No resource dependencies for code generation itself
        ignore=['*/*/zz_generated.deepcopy.go']
    )

    local_resource(
        'Compile Binary',
        binary(),
        deps=COMPILE_DEPS,
        resource_deps=['Generate Code'], # Ensure code generation runs before compile if triggered together
        ignore=[] # No need to ignore here, already handled in generate step if needed
    )

    local_resource('Sample YAML', 'kubectl apply -f ./config/samples/node-role_v1_leaderstatuschecker.yaml', deps=["./config/samples"], resource_deps=[DIRNAME + "-controller-manager"])

    docker_build_with_restart(IMG, '.',
     dockerfile_contents=DOCKERFILE,
     entrypoint='/manager',
     only=['./tilt_bin/manager'],
     live_update=[
           sync('./tilt_bin/manager', '/manager'),
       ]
    )

kubebuilder(
    DOMAIN='bb-noti.io',
    GROUP='node-role',
    VERSION='v1',
    KIND='LeaderStatusChecker',
    IMG='opplieam/bb-noti-operator:v0.0.5',
)

