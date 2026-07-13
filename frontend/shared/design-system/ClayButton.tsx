import type { ButtonHTMLAttributes } from "react";

type ClayButtonProps = ButtonHTMLAttributes<HTMLButtonElement> & {
  variant?: "primary" | "danger" | "ghost";
};

export function ClayButton({ variant = "primary", className, ...rest }: ClayButtonProps) {
  const classes = [
    "clay-button",
    variant === "danger" ? "clay-button--danger" : "",
    variant === "ghost" ? "clay-button--ghost" : "",
    className ?? "",
  ]
    .filter(Boolean)
    .join(" ");
  return <button className={classes} {...rest} />;
}
