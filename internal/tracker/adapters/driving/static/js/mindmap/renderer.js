import { Layout } from "./layout.js";

function getTriageCategory(impact, effort) {
	if (impact === "high" && effort === "low") {
		return { name: "Do It Now", class: "do-it-now" };
	} else if (impact === "high" && effort === "high") {
		return { name: "Honest Work", class: "honest-work" };
	} else if (impact === "low" && effort === "low") {
		return { name: "Snacking", class: "snacking" };
	} else if (impact === "low" && effort === "high") {
		return { name: "Why?", class: "why" };
	}
	return null;
}

function normalizeStatus(status) {
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

export class Renderer {
	constructor(mindmap, linksContainer, nodesContainer) {
		this.mindmap = mindmap;
		this.linksContainer = linksContainer;
		this.nodesContainer = nodesContainer;
	}

	clear() {
		this.linksContainer.textContent = "";
		this.nodesContainer.textContent = "";
	}

	render(goalsMap) {
		this.clear();

		Object.values(goalsMap).forEach((node) => {
			const isRoot = !node.parent || !node.parent.targetId;

			// Draw circuit traces and solder pads to children
			node.children.forEach((child) => {
				let startX, endX;
				if (child.side === "left") {
					startX = node.x;
					endX = child.x + Layout.nodeWidth;
				} else {
					startX = node.x + (isRoot ? Layout.cpuWidth : Layout.nodeWidth);
					endX = child.x;
				}

				const startY =
					node.y + (isRoot ? Layout.cpuHeight : Layout.nodeHeight) / 2;
				const endY = child.y + Layout.nodeHeight / 2;
				const x_mid = (startX + endX) / 2;

				// Orthogonal routing: Horizontal -> Vertical -> Horizontal
				const pathStr = `M ${startX} ${startY} H ${x_mid} V ${endY} H ${endX}`;

				const relType =
					child.parent && child.parent.relationship
						? child.parent.relationship
						: "optional";
				const path = document.createElementNS(
					"http://www.w3.org/2000/svg",
					"path",
				);
				path.setAttribute("d", pathStr);
				path.setAttribute(
					"class",
					`connection status-${normalizeStatus(child.status)} link-${relType}`,
				);
				this.linksContainer.appendChild(path);

				// Place small circular solder pads at connection points and bends
				const pads = [];
				pads.push({ x: startX, y: startY });
				pads.push({ x: endX, y: endY });
				if (startY !== endY) {
					pads.push({ x: x_mid, y: startY });
					pads.push({ x: x_mid, y: endY });
				}

				pads.forEach((p) => {
					const circle = document.createElementNS(
						"http://www.w3.org/2000/svg",
						"circle",
					);
					circle.setAttribute("cx", p.x);
					circle.setAttribute("cy", p.y);
					circle.setAttribute("r", "4");
					circle.setAttribute(
						"class",
						`solder-pad status-${normalizeStatus(child.status)}`,
					);
					this.linksContainer.appendChild(circle);
				});
			});

			// Create HTML foreignObject node card
			const fo = document.createElementNS(
				"http://www.w3.org/2000/svg",
				"foreignObject",
			);

			const width = isRoot ? Layout.cpuWidth : Layout.nodeWidth;
			const height = isRoot ? Layout.cpuHeight : Layout.nodeHeight;

			fo.setAttribute("x", node.x);
			fo.setAttribute("y", node.y);
			fo.setAttribute("width", width);
			fo.setAttribute("height", height);
			fo.setAttribute(
				"class",
				isRoot ? "root-card-wrapper" : "goal-card-wrapper",
			);
			fo.setAttribute("data-goal-id", node.id);

			const statusClass = normalizeStatus(node.status);
			const impactStr = node.impact || "unknown";
			const effortStr = node.effort || "unknown";
			const statusLabel = node.status || "draft";

			if (isRoot) {
				const rootCard = document.createElementNS(
					"http://www.w3.org/1999/xhtml",
					"div",
				);
				rootCard.setAttribute("class", "root-card");
				const rootCardTitle = document.createElementNS(
					"http://www.w3.org/1999/xhtml",
					"div",
				);
				rootCardTitle.setAttribute("class", "root-card-title");
				rootCardTitle.setAttribute("title", node.name);
				rootCardTitle.textContent = node.name;
				rootCard.appendChild(rootCardTitle);

				const rootBadge = document.createElementNS(
					"http://www.w3.org/1999/xhtml",
					"span",
				);
				rootBadge.setAttribute("class", "badge root-badge");
				rootBadge.textContent = "root";
				rootCard.appendChild(rootBadge);

				fo.appendChild(rootCard);
			} else {
				const triage = getTriageCategory(impactStr, effortStr);
				const triageClass = triage ? `triage-${triage.class}` : "";

				const goalCard = document.createElementNS(
					"http://www.w3.org/1999/xhtml",
					"div",
				);
				goalCard.setAttribute(
					"class",
					`goal-card status-${statusClass} ${triageClass}`,
				);
				goalCard.setAttribute("draggable", "true");

				const goalTitle = document.createElementNS(
					"http://www.w3.org/1999/xhtml",
					"div",
				);
				goalTitle.setAttribute("class", "goal-title");
				goalTitle.setAttribute("title", node.name);
				goalTitle.textContent = node.name;
				goalCard.appendChild(goalTitle);

				const goalMeta = document.createElementNS(
					"http://www.w3.org/1999/xhtml",
					"div",
				);
				goalMeta.setAttribute("class", "goal-meta");

				const statusBadge = document.createElementNS(
					"http://www.w3.org/1999/xhtml",
					"span",
				);
				statusBadge.setAttribute("class", `badge status-${statusClass}`);
				statusBadge.textContent = statusLabel;
				goalMeta.appendChild(statusBadge);

				if (triage) {
					const triageBadge = document.createElementNS(
						"http://www.w3.org/1999/xhtml",
						"span",
					);
					triageBadge.setAttribute("class", `badge triage-${triage.class}`);
					triageBadge.textContent = triage.name;
					goalMeta.appendChild(triageBadge);
				}

				if (node.parent && node.parent.targetId) {
					const relType = node.parent.relationship || "optional";
					const linkBadge = document.createElementNS(
						"http://www.w3.org/1999/xhtml",
						"span",
					);
					linkBadge.setAttribute("class", `badge link-badge link-${relType}`);
					linkBadge.textContent = relType;
					goalMeta.appendChild(linkBadge);
				}

				if (impactStr !== "unknown") {
					const impactBadge = document.createElementNS(
						"http://www.w3.org/1999/xhtml",
						"span",
					);
					impactBadge.setAttribute("class", "badge impact-badge");
					impactBadge.textContent = `Imp: ${impactStr}`;
					goalMeta.appendChild(impactBadge);
				}

				if (effortStr !== "unknown") {
					const effortBadge = document.createElementNS(
						"http://www.w3.org/1999/xhtml",
						"span",
					);
					effortBadge.setAttribute("class", "badge effort-badge");
					effortBadge.textContent = `Eff: ${effortStr}`;
					goalMeta.appendChild(effortBadge);
				}

				goalCard.appendChild(goalMeta);
				fo.appendChild(goalCard);
			}

			const card = fo.querySelector(isRoot ? ".root-card" : ".goal-card");

			// Click listener for details drawer
			card.addEventListener("click", () => {
				this.mindmap.drawer.show(node);
			});

			// Drag and drop event listeners for non-root goal cards
			if (!isRoot) {
				card.addEventListener("dragstart", (e) =>
					this.mindmap.dragDrop.handleDragStart(e, node.id),
				);
				card.addEventListener("dragover", (e) =>
					this.mindmap.dragDrop.handleDragOver(e),
				);
				card.addEventListener("dragleave", (e) =>
					this.mindmap.dragDrop.handleDragLeave(e),
				);
				card.addEventListener("dragend", (e) =>
					this.mindmap.dragDrop.handleDragEnd(e),
				);
				card.addEventListener("drop", (e) =>
					this.mindmap.dragDrop.handleDrop(e, node.id),
				);
			}

			this.nodesContainer.appendChild(fo);
		});
	}

	// Rebuild just the card content of an already-positioned node in place.
	refreshCard(node) {
		const fo = this.nodesContainer.querySelector(`[data-goal-id="${node.id}"]`);
		if (!fo) return;

		const isRoot = !node.parent || !node.parent.targetId;
		const statusClass = normalizeStatus(node.status);
		const impactStr = node.impact || "unknown";
		const effortStr = node.effort || "unknown";
		const statusLabel = node.status || "draft";

		const triage = isRoot ? null : getTriageCategory(impactStr, effortStr);
		const triageClass = triage ? `triage-${triage.class}` : "";

		const card = fo.querySelector(isRoot ? ".root-card" : ".goal-card");
		if (!card) return;

		if (isRoot) {
			card.querySelector(".root-card-title").textContent = node.name;
			return;
		}

		// Update card class for triage/status colour changes
		card.setAttribute(
			"class",
			`goal-card status-${statusClass} ${triageClass}`,
		);

		// Rebuild meta badges
		const meta = card.querySelector(".goal-meta");
		meta.textContent = "";

		const statusBadge = document.createElementNS(
			"http://www.w3.org/1999/xhtml",
			"span",
		);
		statusBadge.setAttribute("class", `badge status-${statusClass}`);
		statusBadge.textContent = statusLabel;
		meta.appendChild(statusBadge);

		if (triage) {
			const triageBadge = document.createElementNS(
				"http://www.w3.org/1999/xhtml",
				"span",
			);
			triageBadge.setAttribute("class", `badge triage-${triage.class}`);
			triageBadge.textContent = triage.name;
			meta.appendChild(triageBadge);
		}

		if (impactStr !== "unknown") {
			const impactBadge = document.createElementNS(
				"http://www.w3.org/1999/xhtml",
				"span",
			);
			impactBadge.setAttribute("class", "badge impact-badge");
			impactBadge.textContent = `Imp: ${impactStr}`;
			meta.appendChild(impactBadge);
		}

		if (effortStr !== "unknown") {
			const effortBadge = document.createElementNS(
				"http://www.w3.org/1999/xhtml",
				"span",
			);
			effortBadge.setAttribute("class", "badge effort-badge");
			effortBadge.textContent = `Eff: ${effortStr}`;
			meta.appendChild(effortBadge);
		}
	}
}
