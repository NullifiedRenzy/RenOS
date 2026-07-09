import { healthScoreColor } from "@/lib/utils";

interface HealthScoreProps {
  score: number;
  size?: "sm" | "md" | "lg";
}

export function HealthScore({ score, size = "md" }: HealthScoreProps) {
  const dims = size === "sm" ? 40 : size === "lg" ? 80 : 56;
  const stroke = size === "sm" ? 3 : size === "lg" ? 6 : 4;
  const radius = (dims - stroke) / 2;
  const circ = 2 * Math.PI * radius;
  const offset = circ - (Math.min(Math.max(score, 0), 100) / 100) * circ;
  const color = healthScoreColor(score);

  return (
    <div className="inline-flex items-center gap-2">
      <svg width={dims} height={dims} className="-rotate-90">
        <circle
          cx={dims / 2}
          cy={dims / 2}
          r={radius}
          fill="none"
          stroke="var(--color-surface)"
          strokeWidth={stroke}
        />
        <circle
          cx={dims / 2}
          cy={dims / 2}
          r={radius}
          fill="none"
          stroke={color}
          strokeWidth={stroke}
          strokeDasharray={circ}
          strokeDashoffset={offset}
          strokeLinecap="round"
          className="transition-all duration-700 ease-out"
        />
      </svg>
      <span
        className={
          size === "sm" ? "text-body-small font-medium" :
          size === "lg" ? "text-headline-2 font-bold" :
          "text-headline-4 font-semibold"
        }
        style={{ color }}
      >
        {score}
      </span>
    </div>
  );
}
