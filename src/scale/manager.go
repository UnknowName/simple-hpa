package scale

type Scaler interface {
    ScalePod(namespace, service string, count int)
}