import { Layout } from "./layout.js";
import { Renderer } from "./renderer.js";
import { ZoomPan } from "./zoompan.js";
import { DragDrop } from "./dragdrop.js";

export class MindMap {
	constructor(containerSelector, api, drawer) {
		this.container = document.querySelector(containerSelector);
		this.svg = this.container.querySelector("svg");
		this.workspace = this.svg.querySelector("#workspace");
		this.linksContainer = this.svg.querySelector("#links");
		this.nodesContainer = this.svg.querySelector("#nodes");

		this.api = api;
		this.drawer = drawer;
		this.goalsMap = {};
		// Stable side assignments for direct children of root.
		// Keyed by goal ID → 'left' | 'right'. Written once and never
		// overwritten so drag-drop re-renders don't reshuffle the tree.
		this.rootChildSides = {};

		// Instantiate modules
		this.renderer = new Renderer(
			this,
			this.linksContainer,
			this.nodesContainer,
		);
		this.zoomPan = new ZoomPan(this.container, this.workspace, this.svg);
		this.dragDrop = new DragDrop(this);

		this.initControls();
	}

	initControls() {
		const controls = document.getElementById("controls-panel");
		if (controls) {
			controls
				.querySelector('[title="Zoom In"]')
				.addEventListener("click", () => this.zoomPan.zoomIn());
			controls
				.querySelector('[title="Zoom Out"]')
				.addEventListener("click", () => this.zoomPan.zoomOut());
			controls
				.querySelector('[title="Reset View"]')
				.addEventListener("click", () => this.centerOnRoot());
		}
	}

	centerOnRoot() {
		// Find the main root node
		const root = Object.values(this.goalsMap).find(
			(g) => !g.parent || !g.parent.targetId,
		);
		if (!root) return;

		const rect = this.container.getBoundingClientRect();
		const viewportWidth = rect.width || window.innerWidth;
		const viewportHeight = rect.height || window.innerHeight;

		// Position of the root node center (using CPU width/height since it's the root)
		const targetX = root.x + Layout.cpuWidth / 2;
		const targetY = root.y + Layout.cpuHeight / 2;

		this.zoomPan.scale = 0.85; // Slightly zoomed out to see branch layout
		this.zoomPan.translateX = viewportWidth / 2 - targetX * this.zoomPan.scale;
		this.zoomPan.translateY = viewportHeight / 2 - targetY * this.zoomPan.scale;
		this.zoomPan.updateTransform();
	}

	render(goals, isInitial = false) {
		// Build map
		this.goalsMap = {};
		goals.forEach((g) => {
			this.goalsMap[g.id] = { ...g, children: [] };
		});

		const roots = [];
		Object.values(this.goalsMap).forEach((g) => {
			if (!g.parent || !g.parent.targetId) {
				roots.push(g);
			} else {
				const parent = this.goalsMap[g.parent.targetId];
				if (parent) {
					parent.children.push(g);
				} else {
					roots.push(g);
				}
			}
		});

		// Calculate positions, passing the stable side map so re-renders
		// don't reshuffle root children between left and right.
		Layout.compute(this.goalsMap, roots, this.rootChildSides);

		// Render nodes and links
		this.renderer.render(this.goalsMap);

		// Center on root goal on initial load
		// Use a tiny timeout to ensure viewport/SVG layout size computations are finalized
		if (isInitial) {
			setTimeout(() => this.centerOnRoot(), 30);
		}
	}

	// Patch a single goal's data in the in-memory map and refresh its card
	// without recomputing layout. Use this after field-level updates (impact,
	// effort, member) so the tree positions stay stable.
	patchGoal(updatedGoal) {
		const existing = this.goalsMap[updatedGoal.id];
		if (!existing) return;
		// Merge fields that affect card appearance
		existing.status = updatedGoal.status;
		existing.impact = updatedGoal.impact;
		existing.effort = updatedGoal.effort;
		existing.member = updatedGoal.member;
		this.renderer.refreshCard(existing);
	}
}
