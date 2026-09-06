use rand::Rng;
use std::time::Duration;

pub struct Backoff {
    current: Duration,
    max: Duration,
    multiplier: f64,
    jitter: f64,
}

impl Backoff {
    pub fn new(initial: Duration, max: Duration) -> Self {
        Self {
            current: initial,
            max,
            multiplier: 2.0,
            jitter: 0.2,
        }
    }

    pub fn next_with_jitter(&mut self) -> Duration {
        let base_ns = self.current.as_nanos() as u64;
        self.current = std::cmp::min(
            Duration::from_nanos((self.current.as_nanos() as f64 * self.multiplier) as u64),
            self.max,
        );

        if self.jitter <= 0.0 {
            return Duration::from_nanos(base_ns);
        }

        let range = (base_ns as f64 * self.jitter).ceil() as u64;
        let jitter = rand::thread_rng().gen_range(0..=range);
        Duration::from_nanos(base_ns + jitter)
    }
}
