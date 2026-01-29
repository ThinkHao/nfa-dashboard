CREATE INDEX idx_traffic_rcn_name_time_cov
ON nfa_school_traffic (
  region, cp, school_name, create_time,
  school_id, total_recv, total_send
);