export class Drawer {
	constructor() {
		this.element = document.getElementById("details-drawer");
		this.titleElem = document.getElementById("drawer-title");
		this.descElem = document.getElementById("drawer-description");
		this.statusBadge = document.getElementById("drawer-status-badge");
		this.statusElem = document.getElementById("drawer-status");
		this.impactElem = document.getElementById("drawer-impact");
		this.effortElem = document.getElementById("drawer-effort");
		this.ownerElem = document.getElementById("drawer-owner");
		this.idElem = document.getElementById("drawer-id");
		this.closeBtn = this.element.querySelector(".close-btn");

		this.copyIdBtn = document.getElementById("drawer-copy-id-btn");
		this.copyDescBtn = document.getElementById("drawer-copy-desc-btn");

		this.onUpdate = null;
		this.currentGoal = null;
		this.members = [];

		this.closeBtn.addEventListener("click", () => this.close());
		if (this.statusElem) {
			this.statusElem.addEventListener("change", () => this.handleStatusChange());
		}
		this.impactElem.addEventListener("change", () => this.handleMetricChange());
		this.effortElem.addEventListener("change", () => this.handleMetricChange());
		this.ownerElem.addEventListener("change", () => this.handleMemberChange());
		if (this.descElem) {
			this.descElem.addEventListener("change", () =>
				this.handleDescriptionChange(),
			);
			this.descElem.addEventListener("blur", () => this.handleDescriptionChange());
			this.descElem.addEventListener("keydown", (e) => {
				if ((e.metaKey || e.ctrlKey) && e.key === "Enter") {
					this.descElem.blur();
				}
			});
		}

		if (this.copyIdBtn) {
			this.copyIdBtn.addEventListener("click", () => this.copyId());
		}
		if (this.idElem) {
			this.idElem.addEventListener("click", () => this.copyId());
		}
		if (this.copyDescBtn) {
			this.copyDescBtn.addEventListener("click", () => this.copyDescription());
		}

		this.handleOutsideClick = (e) => {
			if (!this.element.classList.contains("open")) return;
			if (this.element.contains(e.target)) return;
			if (e.target.closest(".goal-card") || e.target.closest(".root-card"))
				return;
			this.close();
		};
		document.addEventListener("click", this.handleOutsideClick);
	}

	async copyToClipboard(text) {
		if (!text) return false;
		if (navigator.clipboard && navigator.clipboard.writeText) {
			try {
				await navigator.clipboard.writeText(text);
				return true;
			} catch {
				// Fall back to execCommand below
			}
		}
		const textArea = document.createElement("textarea");
		textArea.value = text;
		textArea.style.position = "fixed";
		textArea.style.opacity = "0";
		document.body.appendChild(textArea);
		textArea.focus();
		textArea.select();
		let success = false;
		try {
			success = document.execCommand("copy");
		} catch {
			success = false;
		}
		document.body.removeChild(textArea);
		return success;
	}

	showCopyFeedback(btnElem, targetElem, originalBtnText) {
		if (btnElem) {
			btnElem.classList.add("copied");
			const span = btnElem.querySelector("span");
			if (span) span.textContent = "Copied!";
		}
		if (targetElem) {
			targetElem.classList.add("copied");
		}
		setTimeout(() => {
			if (btnElem) {
				btnElem.classList.remove("copied");
				const span = btnElem.querySelector("span");
				if (span && originalBtnText) span.textContent = originalBtnText;
			}
			if (targetElem) {
				targetElem.classList.remove("copied");
			}
		}, 1500);
	}

	async copyId() {
		if (!this.currentGoal || !this.currentGoal.id) return;
		const ok = await this.copyToClipboard(this.currentGoal.id);
		if (ok) {
			this.showCopyFeedback(this.copyIdBtn, this.idElem, "Copy ID");
		}
	}

	async copyDescription() {
		if (!this.currentGoal) return;
		const desc =
			(this.descElem ? this.descElem.value : "") ||
			this.currentGoal.description ||
			"";
		const ok = await this.copyToClipboard(desc);
		if (ok) {
			this.showCopyFeedback(this.copyDescBtn, this.descElem, "Copy");
		}
	}

	setOnUpdate(onUpdate) {
		this.onUpdate = onUpdate;
	}

	setMembers(members) {
		this.members = members || [];
		this._populateMemberOptions();
	}

	_populateMemberOptions() {
		// Preserve the current selection
		const currentValue = this.ownerElem.value;
		this.ownerElem.textContent = "";

		const unassigned = document.createElement("option");
		unassigned.value = "";
		unassigned.textContent = "Unassigned";
		this.ownerElem.appendChild(unassigned);

		this.members.forEach((m) => {
			const opt = document.createElement("option");
			opt.value = m.id;
			opt.textContent = m.name;
			this.ownerElem.appendChild(opt);
		});

		this.ownerElem.value = currentValue;
	}

	async handleDescriptionChange() {
		if (!this.currentGoal || !this.onUpdate) return;
		const newDesc = this.descElem.value;
		const currentDesc = this.currentGoal.description || "";

		if (newDesc === currentDesc) return;

		try {
			await this.onUpdate(this.currentGoal.id, {
				description: newDesc,
			});
			this.currentGoal.description = newDesc;
		} catch (err) {
			alert("Failed to update description: " + err.message);
			this.descElem.value = this.currentGoal.description || "";
		}
	}

	async handleStatusChange() {
		if (!this.currentGoal || !this.onUpdate) return;
		const newStatus = this.statusElem.value;
		const currentNorm = this.normalizeStatus(this.currentGoal.status);

		if (newStatus === currentNorm) return;

		try {
			await this.onUpdate(this.currentGoal.id, {
				status: newStatus,
			});
			this.currentGoal.status = newStatus;
			const status = this.currentGoal.status || "draft";
			this.statusBadge.textContent = status;
			const normalized = this.normalizeStatus(status);
			this.statusBadge.className = `badge status-${normalized}`;
		} catch (err) {
			alert("Failed to update status: " + err.message);
			this.statusElem.value = currentNorm;
		}
	}

	async handleMetricChange() {
		if (!this.currentGoal || !this.onUpdate) return;
		const newImpact = this.impactElem.value;
		const newEffort = this.effortElem.value;

		if (
			newImpact === this.currentGoal.impact &&
			newEffort === this.currentGoal.effort
		) {
			return;
		}

		try {
			await this.onUpdate(this.currentGoal.id, {
				impact: newImpact,
				effort: newEffort,
			});
			this.currentGoal.impact = newImpact;
			this.currentGoal.effort = newEffort;
		} catch (err) {
			alert("Failed to update goal: " + err.message);
			this.impactElem.value = this.currentGoal.impact || "unknown";
			this.effortElem.value = this.currentGoal.effort || "unknown";
		}
	}

	async handleMemberChange() {
		if (!this.currentGoal || !this.onUpdate) return;
		const newMemberId = this.ownerElem.value;
		const currentMemberId = this.currentGoal.member
			? this.currentGoal.member.id
			: "";

		if (newMemberId === currentMemberId) return;

		try {
			await this.onUpdate(this.currentGoal.id, { memberId: newMemberId });
		} catch (err) {
			alert("Failed to assign member: " + err.message);
			this.ownerElem.value = currentMemberId;
		}
	}

	normalizeStatus(status) {
		if (!status) return "draft";
		const s = status.toLowerCase().trim().replace(" ", "-");
		if (
			s === "in-progress" ||
			s === "completed" ||
			s === "cancelled" ||
			s === "open" ||
			s === "draft"
		) {
			return s;
		}
		return "draft";
	}

	show(goal) {
		if (!goal) return;
		this.currentGoal = goal;

		this.titleElem.textContent = goal.name || "";

		if (this.descElem) {
			this.descElem.value = goal.description || "";
		}

		const status = goal.status || "draft";
		this.statusBadge.textContent = status;
		const normalized = this.normalizeStatus(status);
		this.statusBadge.className = `badge status-${normalized}`;
		if (this.statusElem) {
			this.statusElem.value = normalized;
		}

		this.impactElem.value = goal.impact || "unknown";
		this.effortElem.value = goal.effort || "unknown";

		// Populate member select and set current value
		this._populateMemberOptions();
		const memberId = goal.member ? goal.member.id : "";
		this.ownerElem.value = memberId;
		if (memberId) {
			this.ownerElem.classList.remove("muted-text");
		} else {
			this.ownerElem.classList.add("muted-text");
		}

		this.ownerElem.addEventListener(
			"change",
			() => {
				if (this.ownerElem.value) {
					this.ownerElem.classList.remove("muted-text");
				} else {
					this.ownerElem.classList.add("muted-text");
				}
			},
			{ once: true },
		);

		this.idElem.textContent = goal.id || "";
		this.element.classList.add("open");
	}

	close() {
		this.element.classList.remove("open");
	}
}
