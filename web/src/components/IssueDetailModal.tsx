import { useState } from 'react';
import { addDraft } from '../api/client';
import type { Issue } from '../api/client';
import { Modal, Button, LabelBadge } from './ui';

interface IssueDetailModalProps {
  issue: Issue | null;
  isOpen: boolean;
  onClose: () => void;
  onRefresh: () => void;
}

const STATUS_OPTIONS = ['BACKLOG', 'PLANNED', 'DOING', 'BLOCKED', 'DONE'];

export default function IssueDetailModal({ issue, isOpen, onClose, onRefresh }: IssueDetailModalProps) {
  // Editing state
  const [editingField, setEditingField] = useState<string | null>(null);
  const [editTitle, setEditTitle] = useState('');
  const [editDescription, setEditDescription] = useState('');
  const [editEstimate, setEditEstimate] = useState(0);
  const [newComment, setNewComment] = useState('');
  const [saving, setSaving] = useState(false);

  if (!issue) return null;



  const startEditing = (field: string) => {
    setEditingField(field);
    if (field === 'title') setEditTitle(issue.title);
    if (field === 'description') setEditDescription(issue.description || '');
    if (field === 'estimate') setEditEstimate(issue.estimate || 0);
  };

  const cancelEditing = () => setEditingField(null);

  const saveDraft = async (type: string, payload: unknown) => {
    setSaving(true);
    try {
      await addDraft(issue.id, type, payload);
      onRefresh();
    } catch (err) {
      console.error('Failed to save draft:', err);
    } finally {
      setSaving(false);
      setEditingField(null);
    }
  };

  const handleSaveTitle = () => {
    if (editTitle.trim() && editTitle !== issue.title) {
      saveDraft('UPDATE', { title: editTitle.trim() });
    } else {
      cancelEditing();
    }
  };

  const handleSaveDescription = () => {
    if (editDescription !== (issue.description || '')) {
      saveDraft('UPDATE', { description: editDescription });
    } else {
      cancelEditing();
    }
  };

  const handleSaveEstimate = () => {
    if (editEstimate !== (issue.estimate || 0)) {
      saveDraft('UPDATE', { estimate: editEstimate });
    } else {
      cancelEditing();
    }
  };

  const handleStatusChange = (newStatus: string) => {
    if (newStatus !== issue.status) {
      saveDraft('UPDATE', { status: newStatus });
    }
  };



  const handleAddComment = () => {
    if (newComment.trim()) {
      const commentId = `c-${Date.now().toString(36)}`;
      saveDraft('COMMENT', { id: commentId, text: newComment.trim() });
      setNewComment('');
    }
  };

  return (
    <Modal isOpen={isOpen} onClose={onClose} title="" size="2xl" showCloseButton={false}>
      <div className="flex gap-6" style={{ minHeight: '400px' }}>
        {/* Left Column — Main Content */}
        <div className="flex-1 min-w-0 space-y-5">
          {/* Title */}
          <div>
            {editingField === 'title' ? (
              <div className="flex gap-2">
                <input
                  autoFocus
                  value={editTitle}
                  onChange={(e) => setEditTitle(e.target.value)}
                  onKeyDown={(e) => {
                    if (e.key === 'Enter') handleSaveTitle();
                    if (e.key === 'Escape') cancelEditing();
                  }}
                  className="flex-1 text-xl font-bold bg-[var(--color-bg-tertiary)] text-[var(--color-text-primary)] border border-[var(--color-border-default)] rounded-[var(--radius-md)] px-3 py-1.5 outline-none focus:border-[var(--color-accent-primary)]"
                />
                <Button size="sm" onClick={handleSaveTitle} disabled={saving}>Save</Button>
                <Button size="sm" variant="ghost" onClick={cancelEditing}>Cancel</Button>
              </div>
            ) : (
              <h2
                onClick={() => startEditing('title')}
                className="text-xl font-bold text-[var(--color-text-primary)] cursor-pointer hover:text-[var(--color-text-accent)] transition-colors"
                title="Click to edit"
              >
                {issue.title}
              </h2>
            )}
            <div className="flex items-center gap-2 mt-2">
              <span className="text-xs font-mono text-[var(--color-text-muted)]">{issue.id}</span>
              <div className="flex gap-1">
                {issue.labels?.map(label => (
                  <LabelBadge key={label} label={label} />
                ))}
              </div>
              {issue.is_pending && (
                <span className="text-xs px-2 py-0.5 rounded-[var(--radius-sm)] bg-[var(--color-warning-bg)] text-[var(--color-warning)]">
                  Pending changes
                </span>
              )}
            </div>
          </div>

          {/* Description */}
          <div>
            <h3 className="text-xs font-semibold text-[var(--color-text-muted)] uppercase tracking-wider mb-2">Description</h3>
            {editingField === 'description' ? (
              <div className="space-y-2">
                <textarea
                  autoFocus
                  value={editDescription}
                  onChange={(e) => setEditDescription(e.target.value)}
                  onKeyDown={(e) => {
                    if (e.key === 'Escape') cancelEditing();
                  }}
                  rows={5}
                  className="w-full text-sm bg-[var(--color-bg-tertiary)] text-[var(--color-text-primary)] border border-[var(--color-border-default)] rounded-[var(--radius-md)] px-3 py-2 outline-none focus:border-[var(--color-accent-primary)] resize-y"
                />
                <div className="flex gap-2">
                  <Button size="sm" onClick={handleSaveDescription} disabled={saving}>Save</Button>
                  <Button size="sm" variant="ghost" onClick={cancelEditing}>Cancel</Button>
                </div>
              </div>
            ) : (
              <div
                onClick={() => startEditing('description')}
                className="cursor-pointer rounded-[var(--radius-md)] p-3 bg-[var(--color-bg-tertiary)]/50 hover:bg-[var(--color-bg-tertiary)] transition-colors min-h-[60px]"
                title="Click to edit"
              >
                {issue.description ? (
                  <p className="text-sm text-[var(--color-text-secondary)] whitespace-pre-wrap">{issue.description}</p>
                ) : (
                  <p className="text-sm text-[var(--color-text-muted)] italic">Click to add a description…</p>
                )}
              </div>
            )}
          </div>



          {/* Comments */}
          <div>
            <h3 className="text-xs font-semibold text-[var(--color-text-muted)] uppercase tracking-wider mb-3">
              Comments {issue.comments && issue.comments.length > 0 && `(${issue.comments.length})`}
            </h3>

            {/* Existing Comments */}
            {issue.comments && issue.comments.length > 0 && (
              <div className="space-y-3 mb-4">
                {issue.comments.map((comment) => (
                  <div
                    key={comment.id}
                    className="p-3 rounded-[var(--radius-md)] bg-[var(--color-bg-tertiary)] border-l-2 border-[var(--color-accent-primary)]"
                  >
                    <div className="flex items-center justify-between mb-1.5">
                      <span className="text-xs font-medium text-[var(--color-text-accent)]">
                        {comment.created_by}
                      </span>
                      <span className="text-xs text-[var(--color-text-muted)]">
                        {new Date(comment.created_at).toLocaleDateString()}
                      </span>
                    </div>
                    <p className="text-sm text-[var(--color-text-secondary)]">{comment.text}</p>
                  </div>
                ))}
              </div>
            )}

            {/* Add Comment */}
            <div className="space-y-2">
              <textarea
                value={newComment}
                onChange={(e) => setNewComment(e.target.value)}
                placeholder="Add a comment…"
                rows={2}
                className="w-full text-sm bg-[var(--color-bg-tertiary)] text-[var(--color-text-primary)] border border-[var(--color-border-subtle)] rounded-[var(--radius-md)] px-3 py-2 outline-none focus:border-[var(--color-accent-primary)] placeholder:text-[var(--color-text-muted)] resize-none"
                onKeyDown={(e) => {
                  if (e.key === 'Enter' && (e.metaKey || e.ctrlKey)) handleAddComment();
                }}
              />
              <div className="flex justify-end">
                <Button
                  size="sm"
                  onClick={handleAddComment}
                  disabled={!newComment.trim() || saving}
                >
                  Comment
                </Button>
              </div>
            </div>
          </div>
        </div>

        {/* Right Column — Sidebar */}
        <div className="w-52 flex-shrink-0 space-y-5">
          {/* Status */}
          <div>
            <h3 className="text-xs font-semibold text-[var(--color-text-muted)] uppercase tracking-wider mb-2">Status</h3>
            <select
              value={issue.status}
              onChange={(e) => handleStatusChange(e.target.value)}
              disabled={saving}
              className="w-full text-sm bg-[var(--color-bg-tertiary)] text-[var(--color-text-primary)] border border-[var(--color-border-default)] rounded-[var(--radius-md)] px-3 py-2 outline-none focus:border-[var(--color-accent-primary)] cursor-pointer appearance-none"
              style={{
                backgroundImage: `url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='12' height='12' fill='%239ca3af' viewBox='0 0 24 24'%3E%3Cpath d='M7 10l5 5 5-5z'/%3E%3C/svg%3E")`,
                backgroundRepeat: 'no-repeat',
                backgroundPosition: 'right 8px center',
              }}
            >
              {STATUS_OPTIONS.map((s) => (
                <option key={s} value={s}>{s}</option>
              ))}
            </select>
          </div>

          {/* Estimate */}
          <div>
            <h3 className="text-xs font-semibold text-[var(--color-text-muted)] uppercase tracking-wider mb-2">Estimate</h3>
            {editingField === 'estimate' ? (
              <div className="flex gap-1">
                <input
                  autoFocus
                  type="number"
                  min={0}
                  value={editEstimate}
                  onChange={(e) => setEditEstimate(parseInt(e.target.value) || 0)}
                  onKeyDown={(e) => {
                    if (e.key === 'Enter') handleSaveEstimate();
                    if (e.key === 'Escape') cancelEditing();
                  }}
                  className="w-full text-sm bg-[var(--color-bg-tertiary)] text-[var(--color-text-primary)] border border-[var(--color-border-default)] rounded-[var(--radius-md)] px-3 py-2 outline-none focus:border-[var(--color-accent-primary)]"
                />
                <Button size="sm" onClick={handleSaveEstimate} disabled={saving}>
                  <svg className="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}><path strokeLinecap="round" strokeLinejoin="round" d="M5 13l4 4L19 7" /></svg>
                </Button>
              </div>
            ) : (
              <div
                onClick={() => startEditing('estimate')}
                className="p-3 rounded-[var(--radius-md)] bg-[var(--color-bg-tertiary)] cursor-pointer hover:bg-[var(--color-bg-hover)] transition-colors"
                title="Click to edit"
              >
                <span className="text-lg font-bold text-[var(--color-text-primary)]">{issue.estimate || 0}</span>
                <span className="text-xs text-[var(--color-text-muted)] ml-1">pts</span>
              </div>
            )}
          </div>





          {/* Dependencies */}
          {issue.dependencies && issue.dependencies.length > 0 && (
            <div>
              <h3 className="text-xs font-semibold text-[var(--color-text-muted)] uppercase tracking-wider mb-2">Dependencies</h3>
              <div className="space-y-1">
                {issue.dependencies.map((dep, i) => (
                  <div key={i} className="flex items-center gap-1.5 text-xs p-1.5 rounded-[var(--radius-sm)] bg-[var(--color-bg-tertiary)]">
                    <span className="text-[var(--color-text-muted)]">{dep.kind.replace('_', ' ')}</span>
                    <span className="font-mono text-[var(--color-text-accent)]">{dep.target_id}</span>
                  </div>
                ))}
              </div>
            </div>
          )}

          {/* Timestamps */}
          <div className="pt-3 border-t border-[var(--color-border-subtle)] space-y-1.5 text-xs text-[var(--color-text-muted)]">
            <div>
              <span className="font-medium">Created</span><br />
              {new Date(issue.created_at).toLocaleString()}
            </div>
            <div>
              <span className="font-medium">Updated</span><br />
              {new Date(issue.updated_at).toLocaleString()}
            </div>
          </div>
        </div>
      </div>

      {/* Footer */}
      <div className="flex items-center justify-end gap-3 pt-4 mt-4 border-t border-[var(--color-border-subtle)]">
        <Button variant="ghost" onClick={onClose}>Close</Button>
      </div>
    </Modal>
  );
}
