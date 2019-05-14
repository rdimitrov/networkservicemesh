---
apiVersion: apps/v1
kind: Deployment
spec:
  selector:
    matchLabels:
      networkservicemesh.io/app: "icmp-responder"
      networkservicemesh.io/impl: "icmp-responder"
  replicas: 2
  template:
    metadata:
      labels:
        networkservicemesh.io/app: "icmp-responder"
        networkservicemesh.io/impl: "icmp-responder"
    spec:
      affinity:
        podAntiAffinity:
          requiredDuringSchedulingIgnoredDuringExecution:
            - labelSelector:
                matchExpressions:
                  - key: networkservicemesh.io/app
                    operator: In
                    values:
                      - icmp-responder
                  - key: networkservicemesh.io/impl
                    operator: In
                    values:
                      - icmp-responder
              topologyKey: "kubernetes.io/hostname"
      containers:
        - name: icmp-responder-nse
          image: {{ .Values.registry }}/networkservicemesh/test-nse:{{ .Values.tag}}
          imagePullPolicy: {{ .Values.pullPolicy }}
          env:
            - name: NSE_IMAGE
              value: "icmp-responder-nse"
            - name: ADVERTISE_NSE_NAME
              value: "icmp-responder"
            - name: ADVERTISE_NSE_LABELS
              value: "app=icmp-responder"
            - name: TRACER_ENABLED
              value: "true"
            - name: IP_ADDRESS
              value: "10.20.1.0/24"
          resources:
            limits:
              networkservicemesh.io/socket: 1
          command:
          - "/bin/icmp-responder-nse"
metadata:
  name: icmp-responder-nse
  namespace: {{ .Release.Namespace }}
