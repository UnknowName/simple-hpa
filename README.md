# Simple HPA Base Ingress Access Log

## How to Use

## Requirement

- `Kubernetes`
- `NGINX Ingress`

  `Ingress Nginx` Add `ConfigMap` of `log-format-upstream`
    ```bash
    kubectl edit cm/nginx-configuration -n ingress-nginx
    ```

    ```yaml
    disable-access-log: "false"
    access-log-path: "syslog:server=ServerIP:ServerPort"
    log-format-upstream: '{"time_str": "$time_iso8601",
                       "time_msec": $msec,
                       "remote_addr": "$proxy_protocol_addr",
                       "x-forward-for": "$http_x_forwarded_for",
                       "request_time": $request_time,
                       "upstream_response_time": "$upstream_response_time",
                       "upstream_status": $upstream_status,
                       "status": $status,
                       "hostname": "$host",
                       "namespace": "$namespace",
                       "service": "$service_name"}'
    ```
  