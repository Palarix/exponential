import {
  useRef,
  useEffect,
  useLayoutEffect,
  useCallback,
  type ReactNode,
} from "react";
import { createPortal } from "react-dom";
import { Menu, SubMenuTrigger } from "./Menu";
import { computePopoverPosition } from "./popover-utils";

interface SubMenuProps {
  label: string;
  icon?: ReactNode;
  shortcut?: string;
  disabled?: boolean;
  suffix?: ReactNode;
  /** Custom panel content — rendered raw in the flyout */
  renderPanel?: (close: () => void) => ReactNode;
  /** Standard menu items — wrapped in a <Menu> with full keyboard nav */
  children?: ReactNode;
  maxHeight?: string;
  _menuIndex?: number;
  _focused?: boolean;
  _onMouseEnter?: (index: number) => void;
  _subMenuOpen?: boolean;
  _onSubMenuClose?: () => void;
}

const CLOSE_DELAY = 300;

export function SubMenu({
  label,
  icon,
  shortcut,
  disabled,
  suffix,
  renderPanel,
  children,
  maxHeight,
  _menuIndex = 0,
  _focused,
  _onMouseEnter,
  _subMenuOpen,
  _onSubMenuClose,
}: SubMenuProps) {
  const triggerRef = useRef<HTMLDivElement>(null);
  const flyoutRef = useRef<HTMLDivElement>(null);
  const closeTimer = useRef<ReturnType<typeof setTimeout> | null>(null);

  const cancelClose = useCallback(() => {
    if (closeTimer.current) {
      clearTimeout(closeTimer.current);
      closeTimer.current = null;
    }
  }, []);

  const scheduleClose = useCallback(() => {
    cancelClose();
    closeTimer.current = setTimeout(() => {
      closeTimer.current = null;
      _onSubMenuClose?.();
    }, CLOSE_DELAY);
  }, [_onSubMenuClose, cancelClose]);

  useEffect(() => {
    if (!_subMenuOpen) cancelClose();
    return () => cancelClose();
  }, [_subMenuOpen, cancelClose]);

  useLayoutEffect(() => {
    if (!_subMenuOpen) return;
    const anchor = triggerRef.current;
    const flyout = flyoutRef.current;
    if (!anchor || !flyout) return;

    const aRect = anchor.getBoundingClientRect();
    const fRect = flyout.getBoundingClientRect();

    const firstItem = flyout.querySelector("[data-menu-index], [role='menuitem'], [role='menuitemcheckbox']") as HTMLElement;
    const itemOffset = firstItem ? firstItem.getBoundingClientRect().top - fRect.top : 0;

    const { top, left } = computePopoverPosition(
      aRect,
      { width: fRect.width, height: fRect.height },
      "right-start",
      4,
      { width: window.innerWidth, height: window.innerHeight },
    );

    const alignedTop = top - itemOffset;
    const pad = 8;
    const clampedTop = Math.max(pad, Math.min(alignedTop, window.innerHeight - fRect.height - pad));

    flyout.style.top = `${clampedTop}px`;
    flyout.style.left = `${left}px`;
    flyout.style.visibility = "visible";
  });

  const useMenu = !renderPanel && children;

  const handleFlyoutMouseEnter = useCallback(() => {
    cancelClose();
  }, [cancelClose]);

  const handleFlyoutMouseLeave = useCallback(() => {
    scheduleClose();
  }, [scheduleClose]);

  return (
    <>
      <div
        ref={triggerRef}
        onMouseLeave={() => {
          if (_subMenuOpen) scheduleClose();
        }}
      >
        <SubMenuTrigger
          label={label}
          icon={icon}
          shortcut={shortcut}
          disabled={disabled}
          suffix={suffix}
          open={_subMenuOpen}
          focused={_focused}
          menuIndex={_menuIndex}
          onMouseEnter={() => {
            cancelClose();
            _onMouseEnter?.(_menuIndex);
          }}
          onClick={() => _onMouseEnter?.(_menuIndex)}
        />
      </div>
      {_subMenuOpen && _onSubMenuClose &&
        createPortal(
          <div
            ref={flyoutRef}
            data-submenu-flyout
            tabIndex={useMenu ? undefined : -1}
            style={{ position: "fixed", visibility: "hidden" }}
            className={useMenu
              ? "z-[101]"
              : "z-[101] min-w-50 max-w-70 bg-[var(--color-surface-3)] border border-[var(--color-border-elevated)] rounded-[var(--radius-lg)] shadow-[var(--shadow-popover)] overflow-hidden outline-none"
            }
            onMouseEnter={handleFlyoutMouseEnter}
            onMouseLeave={handleFlyoutMouseLeave}
          >
            {useMenu ? (
              <Menu
                onClose={_onSubMenuClose}
                onParentArrowLeft={_onSubMenuClose}
                maxHeight={maxHeight}
                autoFocus
              >
                {children}
              </Menu>
            ) : (
              renderPanel!(_onSubMenuClose)
            )}
          </div>,
          document.body,
        )}
    </>
  );
}

SubMenu._isSubMenu = true as const;
