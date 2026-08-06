export class Drawer {
	constructor() {
		this.element = document.getElementById("details-drawer");
		this.titleElem = document.getElementById("drawer-title");
		this.descElem = document.getElementById("drawer-description");
		this.statusBadge = document.getElementById("drawer-status-badge");
		this.impactElem = document.getElementById("drawer-impact");
		this.effortElem = document.getElementById("drawer-effort");
		this.ownerElem = document.getElementById("drawer-owner");
		this.idElem = document.getElementById("drawer-id");
		this.closeBtn = this.element.querySelector(".close-btn");
		this.onUpdate = null;
		this.currentGoal = null;
		this.members = [];

		this.closeBtn.addEventListener("click", () => this.close());
		this.impactElem.addEventListener("change", () => this.handleMetricChange());
		this.effortElem.addEventListener("change", () => this.handleMetricChange());
		this.ownerElem.addEventListener("change", () => this.handleMemberChange());
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

		if (goal.description) {
			this.descElem.textContent = goal.description;
			this.descElem.classList.remove("muted-text");
		} else {
			this.descElem.textContent = "No description provided.";
			this.descElem.classList.add("muted-text");
		}

		const status = goal.status || "draft";
		this.statusBadge.textContent = status;
		const normalized = this.normalizeStatus(status);
		this.statusBadge.className = `badge status-${normalized}`;

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
