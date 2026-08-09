"use client";

import { useState, useEffect, useCallback } from "react";
import { api } from "@/lib/api-client";
import type { IdentityProvider, OAuth2App, APIToken, IAMRole, IAMRoleAssignment, ApprovalWorkflow, ApprovalRequest, IAMAuditEntry } from "@/lib/types";

export function useIAM() {
  const [providers, setProviders] = useState<IdentityProvider[]>([]);
  const [oauth2Apps, setOAuth2Apps] = useState<OAuth2App[]>([]);
  const [tokens, setTokens] = useState<APIToken[]>([]);
  const [roles, setRoles] = useState<IAMRole[]>([]);
  const [assignments, setAssignments] = useState<IAMRoleAssignment[]>([]);
  const [workflows, setWorkflows] = useState<ApprovalWorkflow[]>([]);
  const [requests, setRequests] = useState<ApprovalRequest[]>([]);
  const [auditLog, setAuditLog] = useState<IAMAuditEntry[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const [pRes, oRes, tRes, rRes, aRes, wRes, reqRes, auditRes] = await Promise.all([
        api.getIdentityProviders(),
        api.getOAuth2Apps(),
        api.getAPITokens(),
        api.getIAMRoles(),
        api.getIAMRoleAssignments(),
        api.getApprovalWorkflows(),
        api.getApprovalRequests(),
        api.getIAMAuditLog(),
      ]);
      setProviders(pRes.data ?? []);
      setOAuth2Apps(oRes.data ?? []);
      setTokens(tRes.data ?? []);
      setRoles(rRes.data ?? []);
      setAssignments(aRes.data ?? []);
      setWorkflows(wRes.data ?? []);
      setRequests(reqRes.data ?? []);
      setAuditLog(auditRes.data ?? []);
    } catch (e) {
      setError(e instanceof Error ? e.message : "Failed to load IAM data");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => { load(); }, [load]);

  const createProvider = useCallback(async (p: Partial<IdentityProvider>) => {
    await api.createIdentityProvider(p);
    await load();
  }, [load]);

  const deleteProvider = useCallback(async (id: string) => {
    await api.deleteIdentityProvider(id);
    await load();
  }, [load]);

  const createOAuth2App = useCallback(async (app: { name: string; redirect_uris?: string[]; scopes?: string[]; description?: string }) => {
    const res = await api.createOAuth2App(app);
    await load();
    return res.client_secret;
  }, [load]);

  const deleteOAuth2App = useCallback(async (id: string) => {
    await api.deleteOAuth2App(id);
    await load();
  }, [load]);

  const createToken = useCallback(async (token: { name: string; scopes?: string[]; expires_at?: string }) => {
    const res = await api.createAPIToken(token);
    await load();
    return res.raw_token;
  }, [load]);

  const deleteToken = useCallback(async (id: string) => {
    await api.deleteAPIToken(id);
    await load();
  }, [load]);

  const createRole = useCallback(async (role: { name: string; display_name?: string; description?: string; permissions?: string[] }) => {
    await api.createIAMRole(role);
    await load();
  }, [load]);

  const deleteRole = useCallback(async (id: string) => {
    await api.deleteIAMRole(id);
    await load();
  }, [load]);

  const createAssignment = useCallback(async (a: { user_id: string; role_id: string; resource_type?: string; resource_id?: string }) => {
    await api.createIAMRoleAssignment(a);
    await load();
  }, [load]);

  const deleteAssignment = useCallback(async (id: string) => {
    await api.deleteIAMRoleAssignment(id);
    await load();
  }, [load]);

  const createWorkflow = useCallback(async (wf: Partial<ApprovalWorkflow>) => {
    await api.createApprovalWorkflow(wf);
    await load();
  }, [load]);

  const deleteWorkflow = useCallback(async (id: string) => {
    await api.deleteApprovalWorkflow(id);
    await load();
  }, [load]);

  const approveRequest = useCallback(async (id: string, comment?: string) => {
    await api.approveRequest(id, comment);
    await load();
  }, [load]);

  const rejectRequest = useCallback(async (id: string, comment?: string) => {
    await api.rejectRequest(id, comment);
    await load();
  }, [load]);

  return {
    providers, oauth2Apps, tokens, roles, assignments, workflows, requests, auditLog,
    loading, error, load,
    createProvider, deleteProvider,
    createOAuth2App, deleteOAuth2App,
    createToken, deleteToken,
    createRole, deleteRole,
    createAssignment, deleteAssignment,
    createWorkflow, deleteWorkflow,
    approveRequest, rejectRequest,
  };
}
