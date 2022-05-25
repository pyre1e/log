CREATE TABLE IF NOT EXISTS logs (
    id          UUID,
    user_id     UUID,
    timestamp   DateTime,
    ip          IPv4
) ENGINE=TinyLog;

CREATE TABLE IF NOT EXISTS events (
    id          UUID,
    log_id      UUID,
    type        String,
    message     String
) ENGINE=TinyLog;
