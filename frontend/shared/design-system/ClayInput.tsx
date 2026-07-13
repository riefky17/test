import type { InputHTMLAttributes } from "react";

type ClayInputProps = InputHTMLAttributes<HTMLInputElement>;

export function ClayInput({ className, ...rest }: ClayInputProps) {
  const classes = ["clay-input", className ?? ""].filter(Boolean).join(" ");
  return <input className={classes} {...rest} />;
}
