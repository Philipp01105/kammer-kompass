"use client";

import { zodResolver } from "@hookform/resolvers/zod";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Building2, KeyRound, LogIn, UserPlus } from "lucide-react";
import { useRouter } from "next/navigation";
import { useState } from "react";
import { useForm, useWatch } from "react-hook-form";
import { z } from "zod";
import { adminApi } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Field } from "@/components/ui/field";
import { Input } from "@/components/ui/input";
import { Select } from "@/components/ui/select";
import { Textarea } from "@/components/ui/textarea";

const loginSchema = z.object({
  email: z.string().min(1, "Bitte Benutzername oder E-Mail eingeben."),
  password: z.string().min(1, "Bitte Passwort eingeben."),
});

const signupSchema = z.object({
  email: z.string().email("Bitte eine gueltige E-Mail eingeben."),
  displayName: z.string().min(2, "Bitte mindestens 2 Zeichen eingeben.").max(100),
  password: z.string().min(10, "Bitte mindestens 10 Zeichen eingeben.").max(256),
  requestRights: z.boolean(),
  requestedRoleTemplateId: z.string().optional(),
  requestedScopeType: z.enum(["global", "state", "ihk"]),
  requestedScopeId: z.string().max(200).optional(),
  proofFileName: z.string().max(255).optional(),
  proofMimeType: z.string().max(100).optional(),
  proofContentBase64: z.string().max(2_000_000).optional(),
  proofNote: z.string().max(2000).optional(),
}).superRefine((value, ctx) => {
  if (!value.requestRights) return;
  if (!value.requestedRoleTemplateId) {
    ctx.addIssue({ code: "custom", path: ["requestedRoleTemplateId"], message: "Bitte Rolle auswählen." });
  }
  if (value.requestedScopeType !== "global" && !value.requestedScopeId) {
    ctx.addIssue({ code: "custom", path: ["requestedScopeId"], message: "Bitte Scope angeben." });
  }
});

type LoginValues = z.infer<typeof loginSchema>;
type SignupValues = z.infer<typeof signupSchema>;

export default function LoginPage() {
  const router = useRouter();
  const queryClient = useQueryClient();
  const [mode, setMode] = useState<"login" | "signup">("login");
  const roles = useQuery({
    queryKey: ["requestable-role-templates"],
    queryFn: adminApi.listRequestableRoleTemplates,
    enabled: mode === "signup",
  });
  const loginForm = useForm<LoginValues>({
    resolver: zodResolver(loginSchema),
    defaultValues: { email: "", password: "" },
  });
  const signupForm = useForm<SignupValues>({
    resolver: zodResolver(signupSchema),
    defaultValues: {
      email: "",
      displayName: "",
      password: "",
      requestRights: false,
      requestedRoleTemplateId: "",
      requestedScopeType: "state",
      requestedScopeId: "",
      proofFileName: "",
      proofMimeType: "",
      proofContentBase64: "",
      proofNote: "",
    },
  });
  const requestRights = useWatch({ control: signupForm.control, name: "requestRights" });
  const requestedScopeType = useWatch({ control: signupForm.control, name: "requestedScopeType" });
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
  const signup = useMutation({
    mutationFn: (body: unknown) => adminApi.register(body),
    onSuccess: (_data, values) => {
      queryClient.removeQueries({ queryKey: ["admin-me"] });
      if (!(values as { requestRights?: boolean }).requestRights) {
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
            Melde dich fuer den internen Bereich an oder erstelle ein normales Nutzerkonto.
            Neue Nutzerkonten erhalten nur die gleichen Rechte wie anonyme Besucher.
          </p>
          <div className="mt-8 rounded-lg border bg-background p-4">
            <div className="flex gap-3">
              <KeyRound className="mt-0.5 h-5 w-5 text-primary" />
              <div>
                <h2 className="font-medium">Admin-Zugang</h2>
                <p className="text-sm text-muted-foreground">
                  Benutzername: <span className="font-mono">super_admin</span>. Das Initialpasswort steht im Backend-Log.
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
              <p className="text-sm text-muted-foreground">Anmelden oder registrieren</p>
            </div>
          </div>

          <div className="mt-6 grid grid-cols-2 rounded-md bg-muted p-1 lg:mt-0">
            <button
              type="button"
              className={mode === "login" ? "rounded bg-background px-3 py-2 text-sm font-medium shadow-sm" : "px-3 py-2 text-sm text-muted-foreground"}
              onClick={() => setMode("login")}
            >
              Anmelden
            </button>
            <button
              type="button"
              className={mode === "signup" ? "rounded bg-background px-3 py-2 text-sm font-medium shadow-sm" : "px-3 py-2 text-sm text-muted-foreground"}
              onClick={() => setMode("signup")}
            >
              Registrieren
            </button>
          </div>

          {mode === "login" ? (
            <form className="mt-6 space-y-4" onSubmit={loginForm.handleSubmit((values) => login.mutate(values))}>
              <div>
                <h2 className="text-2xl font-semibold">Anmelden</h2>
                <p className="mt-1 text-sm text-muted-foreground">Session wird per Secure HTTP-only Cookie gehalten.</p>
              </div>
              <Field label="Benutzername oder E-Mail" error={loginForm.formState.errors.email?.message}>
                <Input autoComplete="username" placeholder="super_admin" {...loginForm.register("email")} />
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
          ) : (
            <form
              className="mt-6 space-y-4"
              onSubmit={signupForm.handleSubmit((values) =>
                signup.mutate({
                  requestRights: values.requestRights,
                  email: values.email,
                  displayName: values.displayName,
                  password: values.password,
                  requestedRoleTemplateId: values.requestRights ? values.requestedRoleTemplateId || undefined : undefined,
                  requestedScopeType: values.requestRights ? values.requestedScopeType : undefined,
                  requestedScopeId:
                    values.requestRights && values.requestedScopeType !== "global"
                      ? values.requestedScopeId || undefined
                      : undefined,
                  proofFileName: values.requestRights ? values.proofFileName || undefined : undefined,
                  proofMimeType: values.requestRights ? values.proofMimeType || undefined : undefined,
                  proofContentBase64: values.requestRights ? values.proofContentBase64 || undefined : undefined,
                  proofNote: values.requestRights ? values.proofNote || undefined : undefined,
                })
              )}
            >
              <div>
                <h2 className="text-2xl font-semibold">Registrieren</h2>
                <p className="mt-1 text-sm text-muted-foreground">
                  Registrierte Nutzer koennen Vorschlaege verfolgen, bekommen aber keine Admin-Rechte.
                </p>
              </div>
              <Field label="E-Mail" error={signupForm.formState.errors.email?.message}>
                <Input type="email" autoComplete="email" {...signupForm.register("email")} />
              </Field>
              <Field label="Anzeigename" error={signupForm.formState.errors.displayName?.message}>
                <Input autoComplete="name" {...signupForm.register("displayName")} />
              </Field>
              <Field label="Passwort" error={signupForm.formState.errors.password?.message}>
                <Input type="password" autoComplete="new-password" {...signupForm.register("password")} />
              </Field>
              <label className="flex items-start gap-3 rounded-md border p-3 text-sm">
                <input type="checkbox" className="mt-1" {...signupForm.register("requestRights")} />
                <span>
                  Ich möchte direkt Rechte beantragen. Dieses Konto wird erst nach Admin-Prüfung aktiviert.
                </span>
              </label>
              {requestRights ? (
                <div className="space-y-4 rounded-md border p-4">
                  <Field label="Gewünschte Rolle" error={signupForm.formState.errors.requestedRoleTemplateId?.message}>
                    <Select {...signupForm.register("requestedRoleTemplateId")}>
                      <option value="">Rolle auswählen</option>
                      {roles.data?.items.map((role) => (
                        <option key={role.id} value={role.id}>
                          {role.name}
                        </option>
                      ))}
                    </Select>
                  </Field>
                  <Field label="Scope" error={signupForm.formState.errors.requestedScopeType?.message}>
                    <Select {...signupForm.register("requestedScopeType")}>
                      <option value="state">Bundesland</option>
                      <option value="ihk">IHK</option>
                      <option value="global">Global</option>
                    </Select>
                  </Field>
                  {requestedScopeType !== "global" ? (
                    <Field label={requestedScopeType === "state" ? "Bundesland" : "IHK-ID"} error={signupForm.formState.errors.requestedScopeId?.message}>
                      <Input {...signupForm.register("requestedScopeId")} />
                    </Field>
                  ) : null}
                  <Field label="Nachweis-Datei, optional" error={signupForm.formState.errors.proofFileName?.message}>
                    <Input
                      type="file"
                      accept="application/pdf,image/jpeg,image/png,image/webp"
                      onChange={async (event) => {
                        const file = event.target.files?.[0];
                        signupForm.setValue("proofFileName", file?.name ?? "");
                        signupForm.setValue("proofMimeType", file?.type ?? "");
                        signupForm.setValue("proofContentBase64", file ? await fileToBase64(file) : "");
                      }}
                    />
                  </Field>
                  <Field label="Nachweis / Begründung, optional" error={signupForm.formState.errors.proofNote?.message}>
                    <Textarea {...signupForm.register("proofNote")} />
                  </Field>
                </div>
              ) : null}
              {signup.error ? <p className="text-sm text-destructive">{signup.error.message}</p> : null}
              {signup.isSuccess && requestRights ? (
                <p className="text-sm text-primary">Deine Registrierung wurde eingereicht und muss von einem Admin geprüft werden.</p>
              ) : null}
              <Button type="submit" disabled={signup.isPending} className="w-full">
                <UserPlus className="h-4 w-4" />
                Konto erstellen
              </Button>
            </form>
          )}
        </div>
      </div>
    </div>
  );
}

function fileToBase64(file: File): Promise<string> {
  return new Promise((resolve, reject) => {
    const reader = new FileReader();
    reader.onload = () => {
      const value = String(reader.result ?? "");
      resolve(value.includes(",") ? value.split(",")[1] ?? "" : value);
    };
    reader.onerror = () => reject(reader.error);
    reader.readAsDataURL(file);
  });
}
