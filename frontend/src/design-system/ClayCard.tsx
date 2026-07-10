import type { HTMLAttributes } from "react";

type ClayCardProps = HTMLAttributes<HTMLDivElement> & {
  flat?: boolean;
};

export function ClayCard({ flat, className, ...rest }: ClayCardProps) {
  const classes = ["clay-card", flat ? "clay-card--flat" : "", className ?? ""]
    .filter(Boolean)
    .join(" ");
  return <div className={classes} {...rest} />;
}
