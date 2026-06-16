"use client";

import { zodResolver } from "@hookform/resolvers/zod";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { Building2, KeyRound, LogIn } from "lucide-react";
import { useRouter } from "next/navigation";
import { useForm } from "react-hook-form";
import { z } from "zod";
import { adminApi } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Field } from "@/components/ui/field";
import { Input } from "@/components/ui/input";

const loginSchema = z.object({
  email: z.string().min(1, "Bitte Benutzername oder E-Mail eingeben."),
  password: z.string().min(1, "Bitte Passwort eingeben."),
});

type LoginValues = z.infer<typeof loginSchema>;

export default function LoginPage() {
  const router = useRouter();
  const queryClient = useQueryClient();
  const loginForm = useForm<LoginValues>({
    resolver: zodResolver(loginSchema),
    defaultValues: { email: "", password: "" },
  });
  const login = useMutation({
    mutationFn: adminApi.login,
    onSuccess: async () => {
      queryClient.removeQueries({ queryKey: ["admin-me"] });
      try {
        await queryClient.fetchQuery({ queryKey: ["admin-me"], queryFn: adminApi.me, retry: false });
        router.push("/admin/dashboard");
      } catch {
        router.push("/");
      }
    },
  });

  return (
    <div className="min-h-screen bg-muted">
      <div className="mx-auto grid min-h-screen max-w-7xl items-center gap-10 px-5 py-10 lg:grid-cols-[1fr_480px]">
        <section className="hidden lg:block">
          <div className="flex h-12 w-12 items-center justify-center rounded-md bg-primary text-primary-foreground">
            <Building2 className="h-6 w-6" />
          </div>
          <h1 className="mt-6 text-5xl font-semibold">KammerKompass</h1>
          <p className="mt-3 max-w-2xl text-lg text-muted-foreground">
            Melde dich mit einem Konto an, das durch den Super-Admin erstellt wurde.
          </p>
          <div className="mt-8 rounded-lg border bg-background p-4">
            <div className="flex gap-3">
              <KeyRound className="mt-0.5 h-5 w-5 text-primary" />
              <div>
                <h2 className="font-medium">Admin-Zugang</h2>
                <p className="text-sm text-muted-foreground">
                  Neue Nutzerkonten werden ausschließlich im Admin-Bereich durch den Super-Admin angelegt.
                </p>
              </div>
            </div>
          </div>
        </section>

        <div className="rounded-lg border bg-background p-7 shadow-sm">
          <div className="flex items-center gap-3 lg:hidden">
            <div className="flex h-10 w-10 items-center justify-center rounded-md bg-primary text-primary-foreground">
              <Building2 className="h-5 w-5" />
            </div>
            <div>
              <h1 className="text-xl font-semibold">KammerKompass</h1>
              <p className="text-sm text-muted-foreground">Anmelden</p>
            </div>
          </div>

          <form className="mt-6 space-y-4 lg:mt-0" onSubmit={loginForm.handleSubmit((values) => login.mutate(values))}>
            <div>
              <h2 className="text-2xl font-semibold">Anmelden</h2>
              <p className="mt-1 text-sm text-muted-foreground">Session wird per Secure HTTP-only Cookie gehalten.</p>
            </div>
            <Field label="Benutzername oder E-Mail" error={loginForm.formState.errors.email?.message}>
              <Input autoComplete="username" placeholder="admin@example.com" {...loginForm.register("email")} />
            </Field>
            <Field label="Passwort" error={loginForm.formState.errors.password?.message}>
              <Input type="password" autoComplete="current-password" {...loginForm.register("password")} />
            </Field>
            {login.error ? <p className="text-sm text-destructive">{login.error.message}</p> : null}
            <Button type="submit" disabled={login.isPending} className="w-full">
              <LogIn className="h-4 w-4" />
              Einloggen
            </Button>
          </form>
        </div>
      </div>
    </div>
  );
}
